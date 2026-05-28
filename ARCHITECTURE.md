# Router Sync Architecture

## Overview

Router Sync is a split-binary system: one Go image runs either as a **central API** (NATS + HTTP only) or as a **per-router agent** (NET_ADMIN, applies kernel routing). NATS JetStream is the source of truth; the web UI is a separate container that calls the API.

Policy routing uses Linux **routing tables** (provisioned by netplan per uplink) plus **`ip rule`** entries (managed by agents per enabled policy).

## Deployment topology

```mermaid
flowchart TB
  subgraph R2["R2 (192.168.2.252)"]
    UI["router-sync-ui :18081"]
    API["router-sync-api --mode=api :18080"]
    NATS["NATS JetStream :4222"]
  end

  subgraph R1["R1"]
    A1["router-sync-agent --mode=agent :18082"]
    K1["Kernel: tables 99/100/200 + ip rules"]
  end

  subgraph R2agent["R2 (agent)"]
    A2["router-sync-agent --mode=agent :18082"]
    K2["Kernel: tables 99/100/200 + ip rules"]
  end

  UI -->|HTTP| API
  API -->|NATS KV| NATS
  A1 -->|NATS KV| NATS
  A2 -->|NATS KV| NATS
  A1 --> K1
  A2 --> K2
```

| Component | Host | Privileges | Ports |
|-----------|------|------------|-------|
| NATS | R2 | — | 4222, 8222 (monitoring) |
| API | R2 | none | 18080 |
| UI | R2 | none | 18081 |
| Agent | R1, R2 | NET_ADMIN, host network | 18082 |

## Process architecture

```mermaid
graph TB
  subgraph binary["cmd/router-sync"]
    MAIN[main.go]
    MAIN -->|mode=api| RUNAPI[runAPI]
    MAIN -->|mode=agent| RUNAGENT[runAgent]
  end

  subgraph api_pkg["internal/api"]
    SERVER[Gin Server]
    HANDLERS[handlers / routers / logging]
    MIGRATOR[provider interface migrator]
    LOGWATCH[API log level watcher]
  end

  subgraph agent_pkg["internal/agent"]
    AGENT[Service]
    AGENT --> WATCH_P[watchProviders]
    AGENT --> WATCH_POL[watchPolicies]
    AGENT --> PUBLISH[publishStateLoop]
    AGENT --> SYNC[periodicSync]
    AGENT --> LOGW[watchLogLevel]
  end

  subgraph router_pkg["internal/router"]
    MGR[Manager]
    MGR --> RULES[ip rule add/del]
    MGR --> SUPPRESS[EnsureSuppressDefaultRule prio 10]
  end

  subgraph state_pkg["internal/state"]
    COLL[Collector linux]
    COLL --> NETLINK[RouteListFiltered all tables]
  end

  subgraph nats_pkg["internal/nats"]
    CLIENT[Client]
    KV1[(router-sync)]
    KV2[(router-sync-state TTL 60s)]
    KV3[(router-sync-logging)]
  end

  RUNAPI --> SERVER
  SERVER --> CLIENT
  RUNAGENT --> AGENT
  AGENT --> MGR
  AGENT --> COLL
  AGENT --> CLIENT
```

## NATS storage layout

```mermaid
graph LR
  subgraph bucket_core["router-sync"]
    P1["provider.Telecom"]
    P2["provider.Starlink"]
    POL1["policies.192.168.2.25"]
    POL2["policies.192.168.2.0_25"]
  end

  subgraph bucket_state["router-sync-state (TTL 60s)"]
    R1["router.r1"]
    R2["router.r2"]
  end

  subgraph bucket_log["router-sync-logging"]
    L1["level.api"]
    L2["level.agent.r1"]
    L3["level.agent.r2"]
  end
```

**Watchers** use subject patterns `providers.>` and `policies.>` (not `.*`) so keys containing dots (policy IDs as IPs/CIDRs) are delivered.

**Writes** use generation + `writer_id` for optimistic concurrency on providers and policies.

## Data models

```mermaid
classDiagram
    class InternetProvider {
        +string ID
        +string Name
        +map Interfaces
        +string Interface deprecated
        +int TableID
        +string Gateway
        +uint64 Generation
        +string WriterID
        +InterfaceForHost(hostname) string
    }

    class RoutingPolicy {
        +string ID
        +string Name
        +string ProviderID
        +bool Enabled
        +uint64 Generation
        +string WriterID
    }

    class RouterState {
        +string Hostname
        +string AgentVersion
        +string LogLevel
        +time Time LastSeen
        +Interface[] Interfaces
        +RoutingTable[] Tables
        +IPRule[] Rules
    }

    RoutingPolicy --> InternetProvider : provider_id
```

## Policy application flow

```mermaid
sequenceDiagram
    participant UI as Web UI
    participant API as API :18080
    participant NATS as NATS KV
    participant A1 as Agent R1
    participant A2 as Agent R2
    participant K as Linux kernel

    UI->>API: PUT /api/v1/policies/192.168.2.25 enabled=true
    API->>NATS: CAS update policy
    NATS-->>A1: policies.> watcher
    NATS-->>A2: policies.> watcher
    A1->>K: ip rule add from 192.168.2.25 lookup 99 prio 2000
    A2->>K: ip rule add from 192.168.2.25 lookup 99 prio 2000
    A1->>NATS: router.r1 state heartbeat
    A2->>NATS: router.r2 state heartbeat
    UI->>API: GET /api/v1/routers
    API->>NATS: list router-sync-state
    API-->>UI: rules + tables per host
```

## Linux routing model

### Tables (netplan — home-router)

Each uplink has a dedicated routing table with a default route, e.g.:

| Provider | Table ID | Interface (example) | Default route |
|----------|----------|---------------------|---------------|
| Telecom | 99 | enp1s0 | via 192.168.4.1 |
| Starlink | 100 | enp2s0 | via 192.168.3.1 |
| Tuenti | 200 | enp3s0 | via 192.168.150.1 |

Defined in `files/r1-netplan.yaml` / `files/r2-netplan.yaml`, applied with `netplan apply` or `make r1-init` / `make r2-init`.

### Rules (agent)

| Priority | Rule | Owner |
|----------|------|-------|
| 10 | `from all lookup main suppress_prefixlength 0` | Agent on start/stop |
| 2000–2032 | `from <src> lookup <table_id>` | Agent per enabled policy |

The **suppress-prefixlength** rule ensures traffic to local subnets uses the main table while only traffic matching the default route falls through to per-source policy rules.

### State collection

`internal/state/collector_linux.go` uses `netlink.RouteListFiltered` with `RT_FILTER_TABLE` and `RT_TABLE_UNSPEC` because `netlink.RouteList` only returns the **main** table. Without this, the UI would show a single table per router.

## API layer

The API server (`internal/api`) has **no** `router.Manager` dependency. It reads and writes NATS only.

| Route group | Responsibility |
|-------------|----------------|
| `/api/v1/providers` | CRUD; normalizes `interfaces` map; migrates legacy `interface` on startup |
| `/api/v1/policies` | CRUD |
| `/api/v1/routers` | List/get router state from `router-sync-state` |
| `/api/v1/logging` | Per-service log levels in `router-sync-logging` |
| `/api/v1/stats` | Aggregates providers, policies, router heartbeats |
| `/api/v1/sync` | No-op (agents sync continuously) |

CORS is enabled for the standalone UI origin.

## Agent layer

`internal/agent/service.go`:

1. `EnsureSuppressDefaultRule()` on start
2. Initial `performFullSync()` — `SyncProviders` + `SyncPolicies`
3. Goroutines: `periodicSync`, `watchProviders`, `watchPolicies`, `publishStateLoop`, `watchLogLevel`
4. On shutdown (via `main`): `CleanupAllRules()` then `RemoveSuppressDefaultRule()`

`internal/router/manager.go` applies policies with priorities 2000–2032, skips duplicate rules, clears conntrack when rules change, and validates one rule per source IP in the managed range.

**Note:** `SetupProvider` currently logs success but does not install routes into provider tables; table defaults come from netplan.

## Web UI

React + Vite + TanStack Query in `web/`. Served by nginx in `router-sync-ui` with runtime `ROUTER_SYNC_API_URL`.

| Page | Data source |
|------|-------------|
| Dashboard | `/health`, `/stats`, `/routers`, `/policies` (enabled-only allocation chart) |
| Routers | `/routers` — interfaces, all tables, rules |
| Devices / Policies | `/policies`, `/providers` |
| Providers | `/providers`, `/routers` (for per-host interface inputs) |
| Settings | `/logging/levels`, per-service `PUT` |

## Metrics

**API** (`:18080/metrics`): HTTP counters, `providers_total`, `policies_total`, `routers_known`, `router_state_age_seconds{hostname}`, `log_level_set_total`.

**Agent** (`:18082/metrics`): `agent_sync_*`, `agent_rules_total`, `agent_routes_total{table}`, `agent_state_publish_*`, `agent_conntrack_cleared_total`.

## Security

- NATS username/password (Ansible `group_vars/secrets.yml`)
- API/UI exposed on LAN only (no auth on HTTP today)
- Agent requires NET_ADMIN and host network
- Config files mode `0640` on routers

## Build and deploy

Single `Dockerfile` builds `./cmd/router-sync`. Ansible playbooks in `home-router`:

- `router-sync-image.yml` — build image on r1 + r2
- `router-sync-api.yml` — API on r2
- `router-sync-agent.yml` — agents on `hosts: routers`
- `router-sync-ui.yml` — UI on r2
- `router-sync.yml` — orchestrator importing the above

## Related docs

- [README.md](README.md) — quick start and API reference
- [BLOG.md](BLOG.md) — narrative overview
- [web/README.md](web/README.md) — UI development
