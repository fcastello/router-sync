# Router Sync: Multi-ISP Policy Routing with a Split API, Agents, and a Web UI

## Introduction

Home networks with more than one internet uplink need more than a default route: you want specific devices or subnets to use Starlink, others on fiber, and everything to keep working when you fail over between two Linux routers. Doing that by hand with `ip rule` and separate routing tables on **each** router does not scale.

**Router Sync** is an open-source stack that centralizes provider and policy configuration in **NATS JetStream**, exposes a **REST API** and **web UI** on one host, and runs a lightweight **agent** on every router that applies policy rules locally and reports live kernel state back.

## My setup

Two Ubuntu routers (**R1** and **R2**) share LAN duties. **R2** also runs:

- **NATS** (source of truth)
- **router-sync API** (`--mode=api`, port 18080)
- **router-sync UI** (port 18081)

Each router runs **router-sync agent** (`--mode=agent`, host network, NET_ADMIN) and three uplinks mapped to routing tables **99** (Telecom), **100** (Starlink), and **200** (Tuenti).

```
  Browser ──► UI :18081 ──► API :18080 ──► NATS :4222
                              ▲              ▲
                              │              │
                         Agent R1       Agent R2
                         ip rule        ip rule
                         tables         tables
                         (netplan)      (netplan)
```

## What changed in the architecture

Earlier versions ran a single service on each router that both served HTTP and touched the kernel. The current design separates concerns:

| Role | Runs where | Touches kernel? |
|------|------------|-----------------|
| API | R2 | No |
| Agent | R1 + R2 | Yes (`ip rule`, state collection) |
| UI | R2 | No |

One Docker image, one binary: `router-sync --mode=api` or `--mode=agent`.

## How routing actually works

### Routing tables (netplan)

Each provider has a **routing table ID** and a **default route** on the correct interface. I define those with **netplan** on each router (separate from Router Sync), for example:

```yaml
enp1s0:
  dhcp4: true
  routes:
    - to: default
      via: 192.168.4.1
      metric: 100
      table: 99
      on-link: true
```

Starlink and Tuenti use tables 100 and 200 on `enp2s0` and `enp3s0`. The agent does **not** install these table routes today; netplan owns them.

### Policy rules (agent)

When I enable a policy for `192.168.2.25` → Telecom, each agent adds:

```text
2000: from 192.168.2.25 lookup 99
```

Disabled policies remove their rules. Changes propagate in a few seconds via NATS watchers (`policies.>` so IDs like `192.168.2.0/25` work).

### The suppress-default rule

Without extra care, policy routing can steal traffic that should stay on the LAN. On agent start, Router Sync installs (if missing):

```text
10: from all lookup main suppress_prefixlength 0
```

Local subnets stay in the **main** table; only traffic that would use the default route continues to the per-source policy rules. The agent removes this rule on clean shutdown.

## Per-router interface names

The same logical provider can use different interface names on each router. In the API and UI:

```json
{
  "name": "Telecom",
  "interfaces": { "r1": "enp1s0", "r2": "enp1s0" },
  "table_id": 99,
  "gateway": "192.168.4.1"
}
```

The Providers page shows one input per router that is currently reporting state. Missing mappings get a warning.

## Web UI

The UI is a separate container so you can redeploy the frontend without touching agents.

| Page | What you see |
|------|----------------|
| **Dashboard** | API health, router cards, traffic allocation (enabled policies only), recent policies |
| **Routers** | Live interfaces, **all** routing tables (main + Telecom/Starlink/Tuenti), IP rules |
| **Policies** | Enable/disable, assign provider |
| **Providers** | Uplinks with per-router interfaces |
| **Settings** | API URL, log level per service (`api`, `agent.r1`, `agent.r2`) |

Open the UI at **http://192.168.2.252:18081**. The API at **:18080** returns JSON only — a browser visit to `:18080/` shows `404`, which is expected.

## Example workflow

**1. Define providers** (once):

```bash
curl -X POST http://192.168.2.252:18080/api/v1/providers \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Starlink",
    "interfaces": {"r1": "enp2s0", "r2": "enp2s0"},
    "table_id": 100,
    "gateway": "192.168.3.1"
  }'
```

**2. Route one laptop through Starlink:**

```bash
curl -X PUT http://192.168.2.252:18080/api/v1/policies/192.168.2.24 \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Pancho2",
    "source_ip": "192.168.2.24",
    "provider_id": "Starlink",
    "enabled": true
  }'
```

Within seconds, both agents install the rule; the Routers page shows it under IP rules and the dashboard allocation chart counts only **enabled** policies.

**3. Turn it off** — set `"enabled": false`; agents remove the rule on the next sync or watcher event.

## NATS as source of truth

Three KV buckets:

- **router-sync** — providers and policies (durable)
- **router-sync-state** — 60s TTL heartbeats with full interface/route/rule snapshots
- **router-sync-logging** — desired log level per service id

If R1’s agent restarts, it reconnects, reinstalls the suppress rule, resyncs all policies, and resumes heartbeats. The API never needs direct SSH to the routers for read-only status.

## Deployment overview

Router Sync does not ship infrastructure-as-code. You bring your own orchestration (Docker Compose, systemd, Kubernetes, etc.). A typical rollout:

1. **Netplan (or equivalent)** on each router — per-uplink routing tables and default routes.
2. **NATS** on a central host with JetStream enabled and credentials configured.
3. **API** — one `router-sync --mode=api` instance pointing at NATS.
4. **Agents** — one `router-sync --mode=agent` per router (`host` network, `NET_ADMIN`, `agent.hostname` matching provider `interfaces` keys).
5. **UI** — `router-sync-ui` container with `ROUTER_SYNC_API_URL` set to the API.

See [README.md — Production deployment](README.md#production-deployment) for netplan and `docker run` examples.

## Monitoring

- API: `curl http://192.168.2.252:18080/metrics`
- Agents: `curl http://r1.fcast.ar:18082/metrics`
- Health: `/health` on both ports

Useful agent metrics: `agent_sync_total`, `agent_rules_total`, `agent_state_publish_total`.

## Lessons learned

1. **Split API and agent** — HTTP and NET_ADMIN should not share a process; central API simplifies the UI and firewall rules.
2. **Watchers need `>` not `*`** — policy IDs are often IP addresses; `policies.*` silently misses them.
3. **Collect all routing tables** — `RouteList` only returns main; the UI needs provider tables too.
4. **Tables vs rules** — netplan for stable uplink defaults; agent for dynamic per-host policy rules.
5. **suppress_prefixlength** — essential for LAN reachability when mixing policy routing with a shared main table.

## Conclusion

Router Sync turns multi-ISP routing from a fragile, per-router shell script into a small control plane: configure in the UI or API, store in NATS, apply everywhere agents run, and inspect live state without SSH.

## Resources

- [README.md](README.md) — install, API, configuration
- [ARCHITECTURE.md](ARCHITECTURE.md) — components, diagrams, data flows
- [web/README.md](web/README.md) — UI development and Docker

---

*Router Sync: one binary, two modes, three NATS buckets, and a dashboard that shows what your routers are actually doing.*
