# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-05-28

[1.0.0]: https://github.com/fcastello/router-sync/compare/v0.1.0...v1.0.0

First stable release of the split **API + agent** architecture with a standalone web UI.

### Added

- **Single binary, two modes** — `--mode=api` (HTTP + NATS only) and `--mode=agent` (NET_ADMIN, host network, kernel routing).
- **NATS JetStream KV** — three buckets: `router-sync` (providers/policies), `router-sync-state` (60s TTL heartbeats), `router-sync-logging` (per-service log levels).
- **Per-router provider interfaces** — `interfaces: {r1: enp1s0, r2: enp2s0}` with automatic migration from legacy `interface` on API startup.
- **Router state API** — `GET /api/v1/routers`, per-host interfaces, routes, and `ip rule` snapshots.
- **Per-service logging** — `GET/PUT /api/v1/logging/level/{service_id}` for `api`, `agent.r1`, `agent.r2`, etc.
- **Agent behavior** — watches providers/policies (`providers.>` / `policies.>`), applies `ip rule` at priority 2000–2032, installs priority-10 `suppress_prefixlength` rule on start.
- **State collector** — reports all routing tables (not only `main`) via `RouteListFiltered`.
- **Web UI** (Vite + React) — Dashboard, Routers, Devices, Policies, Providers, Settings; traffic allocation counts **enabled** policies only.
- **Prometheus metrics** — API on `:18080/metrics`, agent on `:18082/metrics`.
- **Active/active writes** — generation + `writer_id` conflict resolution on provider and policy updates.

### Changed

- Entry point moved to `cmd/router-sync/main.go`; removed monolithic `main.go` and `internal/sync` package (replaced by `internal/agent`).
- API server no longer depends on `router.Manager`; all read paths use NATS KV.
- Default API listen address `:18080`; agent metrics/health on `:18082`.
- Default log level `warn` (was `info` in older configs).
- Dockerfile and Makefile build `./cmd/router-sync`.
- README, ARCHITECTURE.md, and BLOG.md rewritten for the new deployment model.

### Removed

- Combined “sync service on every router that also serves HTTP” deployment model.

## [0.1.0] - 2025-07-11

[0.1.0]: https://github.com/fcastello/router-sync/compare/v0.0.2...v0.1.0

### Added

- Linux installer with systemd unit.

### Changed

- Release tooling and contributing guide updates.

## [0.0.2] - 2025-07-11

[0.0.2]: https://github.com/fcastello/router-sync/compare/v0.0.1...v0.0.2

### Changed

- Changelog and release workflow updates.

## [0.0.1] - 2025-07-11

[0.0.1]: https://github.com/fcastello/router-sync/releases/tag/v0.0.1

### Added

- Initial router synchronization service with NATS.io integration, REST API, and policy-based routing.
