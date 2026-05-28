# Router Sync UI

Standalone web dashboard for [router-sync](../README.md). Built with Vite, React, Tailwind CSS, and TanStack Query.

## Development

```bash
cd web
npm install
npm run dev
```

By default, dev uses `.env.development.local` to proxy `/api` and `/health` to R2 at `http://192.168.2.252:18080`. Override in that file or:

```bash
VITE_API_PROXY=http://192.168.2.252:18080 npm run dev
```

Or set the API URL in **Settings** (stored in `localStorage`).

## Production build

```bash
npm run build
npm run preview
```

## Docker

Build and run pointing at your API:

```bash
docker build -t router-sync-ui:latest ./web
docker run --rm -p 8080:80 \
  -e ROUTER_SYNC_API_URL=http://192.168.2.252:18080 \
  router-sync-ui:latest
```

Open http://localhost:8080

## Deploy on R2 (Ansible)

From `home-router`:

```bash
make r2-router-sync-ui
```

Serves the UI on port **18081** with `ROUTER_SYNC_API_URL` set to the R2 management IP API (`:18080`).

## Features

| View | Description |
|------|-------------|
| **Dashboard** | API health, provider uplinks, policy allocation chart, sync stats |
| **Devices** | Policy list with local friendly names and tags |
| **Policies** | Sentence-style policy builder, enable/disable toggles (optimistic) |
| **Providers** | CRUD for internet uplinks |
| **Settings** | Runtime API base URL |

The UI polls health every 5s and stats every 10s.
