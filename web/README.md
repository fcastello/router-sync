# Router Sync UI

Standalone web dashboard for [router-sync](../README.md). Built with Vite, React, Tailwind CSS, and TanStack Query.

## Development

```bash
cd web
npm install
npm run dev
```

By default, dev proxies `/api` and `/health` to R2 at `http://192.168.2.252:18080`. Override in `.env.development.local` or:

```bash
VITE_API_PROXY=http://192.168.2.252:18080 npm run dev
```

You can also set the API base URL in **Settings** (stored in `localStorage`).

## Production build

```bash
npm run build
npm run preview
```

## Docker

```bash
docker build -t router-sync-ui:latest ./web
docker run --rm -p 18081:80 \
  -e ROUTER_SYNC_API_URL=http://192.168.2.252:18080 \
  router-sync-ui:latest
```

## Production deploy

Build and run the UI container on any host that can reach the API (often the same machine as the API):

```bash
docker build -t router-sync-ui:latest .
docker run -d --name router-sync-ui -p 18081:80 \
  -e ROUTER_SYNC_API_URL=http://<api-host>:18080 \
  router-sync-ui:latest
```

Open **http://&lt;host&gt;:18081**. Set `ROUTER_SYNC_API_URL` to the Router Sync API base URL (no trailing slash). See [README — Production deployment](../README.md#production-deployment) for the full stack (NATS, API, agents, netplan).

## Pages

| Page | Description |
|------|-------------|
| **Policies** (default) | Policy builder, favorites, enable/disable (optimistic updates) |
| **Dashboard** | API health, online routers, provider→interface badges, traffic allocation chart (**enabled policies only**), recent policies |
| **Routers** | Per-router live state: interfaces, **all routing tables** (main + provider tables by name), IP rules, agent version, online dot |
| **Devices** | Same policies as Policies, with browser-local tags |
| **Providers** | CRUD uplinks; one interface field per discovered router; warnings for missing mappings |
| **Settings** | API base URL; runtime log level per service (`api`, `agent.r1`, `agent.r2`) |

Polling: health ~5s, stats/routers ~10s.

## Notes

- Open **http://&lt;router&gt;:18081** for the UI. Port **18080** is the API only (no HTML at `/`).
- Traffic allocation counts only policies with `enabled: true`.
- Router table names (e.g. `Telecom (#99)`) come from provider `table_id` when the agent reports routes in that table.
