# Router Sync

A Go-based router synchronization stack that manages internet providers and routing policies using NATS JetStream as the source of truth. It enables policy-based routing across multiple routers in a LAN environment.

## Architecture (split-binary)

The same binary runs in one of two modes selected at runtime by `--mode`:

| Mode | Where | Network | Responsibilities |
|------|-------|---------|------------------|
| **`--mode=api`** | R2 (or any host) | Published port `:18080`, no NET_ADMIN | REST API, Swagger, metrics; reads/writes NATS only |
| **`--mode=agent`** | Every router (R1 + R2) | `network_mode: host`, NET_ADMIN | Watches NATS, applies `ip rule`, heartbeats `RouterState` every 5s |

A separate **web UI** container on R2 (`:18081`) talks only to the API.

```
                          ┌──────────────────────┐
       browser  ────►     │  router-sync UI      │
                          │  (R2, port 18081)    │
                          └──────────┬───────────┘
                                     │ HTTP
                          ┌──────────▼───────────┐
                          │  router-sync API     │
                          │  --mode=api          │
                          │  (R2, port 18080)    │
                          └──────────┬───────────┘
                                     │ NATS (auth)
                          ┌──────────▼───────────┐
                          │  NATS JetStream      │
                          │  (R2, port 4222)     │
                          │  buckets:            │
                          │    router-sync       │
                          │    router-sync-state │
                          │    router-sync-logging
                          └──┬───────────────┬───┘
                             │               │
              ┌──────────────▼───┐  ┌────────▼─────────┐
              │ Agent on R1      │  │ Agent on R2      │
              │ --mode=agent     │  │ --mode=agent     │
              │ NET_ADMIN, host  │  │ NET_ADMIN, host  │
              │ :18082/metrics   │  │ :18082/metrics   │
              └──────────────────┘  └──────────────────┘
```

### NATS KV buckets

| Bucket | TTL | Keys | Purpose |
|--------|-----|------|---------|
| `router-sync` | none | `provider.{id}`, `policy.{id}` | Providers and policies (source of truth) |
| `router-sync-state` | 60s | `router.{hostname}` | Agent heartbeats: interfaces, routes, rules |
| `router-sync-logging` | none | `level.{service_id}` | Runtime log levels (`api`, `agent.r1`, …) |

### What the agent does on each router

1. **On start** — installs priority-10 rule: `from all lookup main suppress_prefixlength 0` (LAN traffic stays in main; only default-route traffic falls through to policy rules). Skips if already present.
2. **Watches** providers and policies in NATS (`policies.>` / `providers.>` so dotted IDs like `192.168.2.25` match).
3. **Applies** enabled policies as `ip rule` entries at priority 2000–2032 (`from <src> lookup <table_id>`).
4. **Publishes** full router state every 5s (all routing tables via netlink, not just `main`).
5. **On stop** — removes managed policy rules and the suppress-default rule.

Provider **routing tables** (default routes per uplink) must exist on each router before policies work — typically via **netplan**, NetworkManager, or static `ip route` configuration (see [Production deployment](#production-deployment)). The agent owns **`ip rule` policy entries** only; it does not install per-uplink table routes today.

## Features

- **Per-router interface mapping** — `interfaces: {r1: enp1s0, r2: enp2s0}` on each provider; legacy `interface` auto-migrated on API startup.
- **Policy-based routing** — source IP or CIDR → provider routing table; agents on all routers apply rules locally.
- **Live router state in the UI** — interfaces, all routing tables (main + provider), `ip rule` list, online indicator.
- **Runtime log levels per service** — `api`, `agent.r1`, `agent.r2` via API and Settings page.
- **Web UI** — Dashboard, Routers, Devices, Policies, Providers, Settings ([`web/README.md`](web/README.md)).
- **Prometheus** — API `:18080/metrics`, agent `:18082/metrics`.
- **Active/active writes** — generation + `writer_id` conflict resolution on provider/policy updates.
- **Single Docker image** for API and agent (`router-sync:latest`).

## Quick Start

### Build

```bash
make build                    # ./build/router-sync
make run-api                  # API mode locally
make run-agent                # agent mode (needs NET_ADMIN on Linux)
make docker-build
```

### Run API locally

```bash
./build/router-sync --mode=api -config config.yaml
curl http://localhost:18080/health
curl http://localhost:18080/api/v1/routers
```

### Run agent locally (Linux only)

```bash
sudo ./build/router-sync --mode=agent -config config.yaml
curl http://localhost:18082/health
```

### Production deployment

Deploy components in this order:

1. **Routing tables on every router** — Create one Linux routing table per uplink, each with a default route on the correct interface. Provider `table_id` values in the API must match these table numbers. The agent will not create these routes for you.
2. **NATS JetStream** — Run NATS on a host reachable from the API and all agents. Use authentication in production. KV buckets are created automatically on first connect.
3. **API** (`--mode=api`) — One central instance; configure NATS URL/credentials and `api.address` (default `:18080`).
4. **Agent** (`--mode=agent`) — One instance per router that enforces policies. Requires **host network**, **NET_ADMIN**, and `agent.hostname` matching keys in each provider's `interfaces` map (e.g. `router-a`, `router-b`).
5. **Web UI** — Separate container or static site; point `ROUTER_SYNC_API_URL` at the API (see [`web/README.md`](web/README.md)).

#### Example: netplan per-uplink tables

On each router, define default routes in provider-specific tables (IDs must match API `table_id`):

```yaml
# /etc/netplan/99-router-sync.yaml (example — adjust interfaces and gateways)
network:
  version: 2
  ethernets:
    wan-telecom:
      dhcp4: true
      routes:
        - to: default
          via: 192.168.4.1
          table: 99
          on-link: true
    wan-starlink:
      dhcp4: true
      routes:
        - to: default
          via: 192.168.3.1
          table: 100
          on-link: true
```

Apply with `sudo netplan apply`. Repeat equivalent configuration on every router that participates.

#### Example: API container

```bash
docker run -d --name router-sync-api \
  -p 18080:18080 \
  -v /etc/router-sync/config.yaml:/etc/router-sync/config.yaml:ro \
  router-sync:latest \
  --mode=api -config /etc/router-sync/config.yaml
```

`config.yaml` for the API should set `mode: api`, NATS `urls` / credentials, and `api.address: ":18080"`.

#### Example: agent container

```bash
docker run -d --name router-sync-agent \
  --network host --cap-add NET_ADMIN \
  -v /etc/router-sync/config.yaml:/etc/router-sync/config.yaml:ro \
  router-sync:latest \
  --mode=agent -config /etc/router-sync/config.yaml
```

Agent `config.yaml` must set `mode: agent`, the same NATS settings, and `agent.hostname` to this machine's router id (used in provider `interfaces` maps). Health and metrics: `:18082`.

#### Example: UI container

```bash
docker build -t router-sync-ui:latest ./web
docker run -d --name router-sync-ui -p 18081:80 \
  -e ROUTER_SYNC_API_URL=http://<api-host>:18080 \
  router-sync-ui:latest
```

After deployment:

- **UI**: `http://<api-host>:18081`
- **API**: `http://<api-host>:18080` (JSON only — no HTML at `/`)
- **Swagger**: `http://<api-host>:18080/swagger/index.html`

Release binaries and install scripts are on [GitHub Releases](https://github.com/fcastello/router-sync/releases). For systemd-based installs without Docker, see [`scripts/README.md`](scripts/README.md).

## Configuration

Example `config.yaml` (API mode):

```yaml
mode: api
log_level: warn

nats:
  urls:
    - "nats://192.168.2.252:4222"
  username: "router_sync"
  password: "your-password"
  cluster_id: "router-sync-cluster"
  client_id: "router-sync-api"
  writer_id: "api"

api:
  address: ":18080"

sync:
  interval: 30s

agent:
  hostname: "r1"              # agent mode only
  metrics_address: ":18082"
  state_publish_interval: 5s
```

Environment overrides: `ROUTER_SYNC_MODE`, `ROUTER_SYNC_LOG_LEVEL`, `ROUTER_SYNC_NATS_URL`, `ROUTER_SYNC_AGENT_HOSTNAME`, etc. (see `internal/config/config.go`).

## API

Base URL: `http://<host>:18080`

| Area | Endpoints |
|------|-----------|
| Health | `GET /health` |
| Metrics | `GET /metrics` |
| Swagger | `GET /swagger/index.html` |
| Providers | `GET/POST /api/v1/providers`, `GET/PUT/DELETE /api/v1/providers/{id}` |
| Policies | `GET/POST /api/v1/policies`, `GET/PUT/DELETE /api/v1/policies/{id}` |
| Routers | `GET /api/v1/routers`, `GET /api/v1/routers/{hostname}`, `.../interfaces`, `.../routes`, `.../rules` |
| Logging | `GET /api/v1/logging/levels`, `GET/PUT /api/v1/logging/level/{service_id}` |
| Stats | `GET /api/v1/stats` |
| Sync | `POST /api/v1/sync` (no-op; agents sync continuously) |

**CIDR policy IDs in URLs** — use underscore instead of slash: `192.168.2.0_25` for `192.168.2.0/25`.

### Create provider (per-router interfaces)

```bash
curl -X POST http://192.168.2.252:18080/api/v1/providers \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Telecom",
    "interfaces": {"r1": "enp1s0", "r2": "enp1s0"},
    "table_id": 99,
    "gateway": "192.168.4.1",
    "description": "Primary uplink"
  }'
```

### Enable a policy

```bash
curl -X PUT http://192.168.2.252:18080/api/v1/policies/192.168.2.25 \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Pancho",
    "source_ip": "192.168.2.25",
    "provider_id": "Telecom",
    "enabled": true
  }'
```

Agents pick up changes within a few seconds via NATS watchers.

## Data models

### InternetProvider

```json
{
  "id": "Telecom",
  "name": "Telecom",
  "interfaces": { "r1": "enp1s0", "r2": "enp1s0" },
  "table_id": 99,
  "gateway": "192.168.4.1",
  "description": "Primary internet connection",
  "generation": 2,
  "writer_id": "api"
}
```

### RoutingPolicy

Policy `id` is the source IP or CIDR (e.g. `192.168.2.25`, `192.168.2.0/25`).

```json
{
  "id": "192.168.2.25",
  "name": "Pancho",
  "provider_id": "Telecom",
  "enabled": true,
  "generation": 6,
  "writer_id": "api"
}
```

### RouterState (from agent heartbeat)

```json
{
  "hostname": "r1",
  "agent_version": "dev",
  "log_level": "warning",
  "last_seen": "2026-05-28T18:45:00Z",
  "interfaces": [{ "name": "enp1s0", "up": true, "addresses": ["192.168.4.6/24"] }],
  "tables": [{ "id": 99, "name": "Telecom", "routes": [{ "dst": "default", "gateway": "192.168.4.1" }] }],
  "rules": [{ "priority": 10, "from": "all", "table": 254 }, { "priority": 2000, "from": "192.168.2.25", "table": 99 }]
}
```

## Monitoring

### API metrics (`:18080/metrics`)

- `http_requests_total`, `http_request_duration_seconds`
- `providers_total`, `policies_total`
- `routers_known`, `router_state_age_seconds{hostname}`
- `log_level_set_total`

### Agent metrics (`:18082/metrics`)

- `agent_sync_total`, `agent_sync_duration_seconds`
- `agent_rules_total`, `agent_routes_total{table}`
- `agent_state_publish_total`, `agent_state_publish_errors_total`
- `agent_conntrack_cleared_total`

## Project structure

```
router-sync/
├── cmd/router-sync/main.go   # --mode dispatch
├── internal/
│   ├── agent/                # NATS watchers, sync loop, state publisher
│   ├── api/                  # Gin HTTP server
│   ├── config/
│   ├── logging/              # per-service runtime levels
│   ├── metrics/
│   ├── models/
│   ├── nats/                 # three KV buckets, watchers
│   ├── router/               # ip rule manager (agent)
│   └── state/                # netlink collector (linux build tag)
├── web/                      # React UI
├── Dockerfile                # single image, API + agent
├── ARCHITECTURE.md
├── BLOG.md
└── Makefile
```

## Development

```bash
make build
make test
make run-api
make run-agent          # Linux + root for netlink
make ui-install && make ui-dev
make docker-build
```

See [`ARCHITECTURE.md`](ARCHITECTURE.md) for component diagrams and data flows.

## Troubleshooting

| Issue | Check |
|-------|--------|
| `404` at `http://host:18080/` | Expected — use `:18081` for UI or `/health`, `/api/v1/*` for API |
| Policy not applied on router | Agent logs; `curl :18082/health`; NATS connectivity from router |
| Provider table empty | Netplan routes (`table: 99` etc.) — agent does not install table routes yet |
| Router missing in UI | Agent running? `GET /api/v1/routers` — state TTL is 60s |
| Watcher slow | Fixed: watchers use `policies.>` not `policies.*` for dotted policy IDs |

Default log level is **warn**. Set per-service via Settings or `PUT /api/v1/logging/level/agent.r1`.

## License

MIT License — see LICENSE file.
