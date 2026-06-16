# Changelog

All notable changes to Embrionix Dashboard are documented here.  
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Unreleased]

## [0.2.0] — 2026-06-16

### Added

**Comprehensive EM6 monitoring** — the poll now covers every health/telemetry
endpoint on the device (see [API.md → EM6 endpoint coverage](API.md#em6-endpoint-coverage)):
- `/self/diag/refclk` — detailed PTP status: lock state (decoded), offset from
  master, mean delay, sync/delay-request counters, lock events
- `/self/firmware` — firmware bank slots (slot, version, active, default)
- `/self/license` — licensed feature map
- `/self/diag/ethernet` — control-plane TX/RX packet counters and RX errors
- `/self/diag/common` — video bandwidth usage, watchdog, IPv4 packet drops
- `/self/interfaces` — per-interface (e1/e2) IP, gateway, DHCP, VLAN
- `/lldp` — discovered neighbour (chassis, remote port, TTL)
- `/telemetry/devices` — media-flow packet counters and validity
- `/sdi` — SDI operating bit rate

**Backend**
- Dual-path reachability: independent L4 (TCP-connect) probe of Red and Blue
  management IPs each poll cycle, stored on the device and every `PollResult`
  (`reachable_red`, `reachable_blue`); gated by `polling.icmp_enabled`
- Status engine now factors PTP lock, ethernet RX errors, and video-bandwidth
  health into warnings, in addition to alarms and the >75 °C critical threshold
- `PollResult` extended with `ptp_locked`, `ptp_offset`, `reachable_red/blue`
- Fleet-wide alarm endpoint: `GET /api/v1/alarms`
- Daily history-pruning background job (`polling.history_retention_days`, default 30)
- Unit tests for PTP status decoding, status derivation, and fleet-alarm aggregation

**Frontend**
- Fleet-wide alarm panel on the dashboard (click-through to the device)
- Auto-refresh countdown indicator on the dashboard
- Device detail enriched: PTP/refclk card, system-health card, firmware banks,
  per-interface (e1/e2) config, LLDP neighbour, control-plane ethernet counters,
  media-flow table, SDI bit rate, and a PTP-offset trend chart
- Dual-path reachability dots on the Overview network panel
- Toast notifications for device create/update/delete and on-demand polls
- Keyboard shortcut: press **N** to add a device
- API status indicator now verifies real connectivity via `/health`

**Infrastructure**
- Code-splitting: recharts, React, and React Query are split into separate
  vendor chunks; heavy routes are lazy-loaded (initial bundle no longer ships
  the chart library)
- Google Fonts moved to `index.html` (`<link>`), removing the PostCSS `@import`
  warning; page `<title>` and meta description set
- Vitest added with a unit test; `npm test` wired into CI

## [0.1.0] — 2026-06-16

### Added

**Backend**
- Go backend with Gin framework, GORM ORM, and pure-Go SQLite (`glebarez/sqlite`)
- Clean architecture: `cmd/`, `internal/api/`, `internal/services/`, `internal/repositories/`, `internal/models/`, `pkg/`
- `Device` model — full inventory fields (name, location, rack, serial, model, firmware, dual IP, tags, notes, monitoring toggle)
- `PollResult` model — time-series snapshots: temp, fan, core voltage, port SFP TX/RX power, response time
- `AppSetting` model — key/value config store
- `EmsfpClient` — HTTP client for the emSFP REST API; polls `self/information`, `self/ipconfig`, `self/system`, `telemetry/node`, `telemetry/ports`, and `port/{id}` DDM
- `PollingService` — concurrent background poller with configurable interval, in-memory state cache, alarm detection
- REST API: `GET/POST /api/v1/devices`, `GET/PUT/DELETE /api/v1/devices/:id`
- Monitoring endpoints: `/api/v1/devices/:id/history`, `/api/v1/devices/:id/poll`, `/api/v1/devices/:id/reachability`
- Dashboard summary endpoint: `GET /api/v1/summary`
- Settings endpoints: `GET/PUT /api/v1/settings/:key`
- Health endpoint: `GET /health`
- Structured logging via Zap (console + JSON file)
- Configuration via Viper (YAML file + `EMB_` environment variable overrides)
- SQLite WAL mode, WAL-safe single-writer connection pool

**Frontend**
- React 18 + TypeScript + Vite + Tailwind CSS v3
- Dark-mode-first NOC/SOC colour scheme (`surface`, `brand` palette)
- Dashboard page — card view and table view, filter by status, summary counters
- Device cards — status dot, health metrics, dual-IP status, SFP TX/RX, last poll time
- Device detail page — Overview, Interfaces, SFP Modules, Monitoring (Recharts charts + history table), Logs tabs
- Devices inventory page — full CRUD with search, add/edit modal form, delete confirmation
- Monitoring page — fleet-wide table sorted by severity with bar chart distribution
- Settings page — device management, polling config, backup/restore placeholder, about/roadmap
- React Query for server state (30 s auto-refresh for device list, 60 s for history)
- Vite dev-server proxy to Go backend

**Infrastructure**
- GitHub Actions CI: Go vet + test + build, TypeScript type-check + Vite build
- GitHub Actions release: multi-platform binaries (Linux amd64/arm64, Windows amd64, macOS amd64/arm64) + frontend bundled
- Issue templates: bug report, feature request
- PR template with Embrionix API verification checklist
- Dockerfile (multi-stage) + docker-compose.yml
- `.gitignore` for Go, Node.js, SQLite data

**Documentation**
- `README.md` — quick start, configuration, project structure
- `ARCHITECTURE.md` — system design, layer diagram, data flow, DB schema, security notes
- `API.md` — full REST endpoint reference
- `ROADMAP.md` — 5-phase feature plan
- `INSTALLATION.md` — binary, Docker, source, Windows service, systemd
- `CONTRIBUTING.md` — dev setup, workflow, API validation rules
- `ISSUES.md` — known limitations and unsupported API features
