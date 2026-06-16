# Changelog

All notable changes to Embrionix Dashboard are documented here.  
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Unreleased]

### Changed — minimal-impact polling
- **Tiered polling**: dynamic health (~6 GETs) is fetched every cycle; the static
  / heavy endpoints (firmware, license, interfaces, LLDP, media flows, SDI, and
  per-port SFP DDM) only every `polling.full_every` cycle (default 10) and are
  carried forward in between. Cuts steady-state device requests ~17 → ~6 (~58 %).
- **HTTP keep-alive within a poll**: the many GETs of one poll reuse a single TCP
  connection (was one connection per request) — ~88 % fewer handshakes on the
  device's embedded server.
- **Bounded fleet concurrency** (`polling.max_concurrent_polls`, default 8) so a
  large fleet no longer bursts every cycle.
- Alarm derivation moved to a single pass over the final data, so light polls
  stay correct.
- New [PERFORMANCE.md](PERFORMANCE.md) quantifies network/device impact and tuning.

### Added — backlog items
- **Ansible inventory export** (`GET /api/v1/export/ansible`) — devices as Ansible
  dynamic-inventory JSON (group `emsfp`); download button in Settings → Backup.
- **Keyboard shortcuts** — `g`+`d/v/m/s` to navigate, `?` for a help overlay.
- **Auth-aware downloads** — config snapshot, DB backup, history CSV and Ansible
  inventory now download with the bearer token attached (fixes downloads when
  auth is enabled).

### Added — Phase 5 (authentication, RBAC & user management)
- Optional auth, **disabled by default** (`auth.enabled: false`) so existing
  deployments keep running with no login. Enabling requires a `jwt_secret` and
  seeds an admin on first start (password from config, or generated + logged once).
- Local accounts (bcrypt) and JWT bearer auth; optional static `X-API-Key`
  (admin-equivalent) for integrations.
- RBAC with three roles enforced server-side: **viewer** (read), **operator**
  (+ device writes / config / actions), **admin** (+ user management). Routes are
  grouped by privilege; the API is the source of truth (403 on under-privilege).
- Endpoints: `POST /auth/login`, `GET /auth/me`, and admin `GET/POST/PUT/DELETE /users`.
- Frontend: login screen, auth context, token handling with 401 → re-login,
  username/role + sign-out in the top bar, role-gated controls (Add Device,
  config editing, device actions, Poll Now), and a Settings → Users & Access tab.
- Tests for role ranking, password hashing, and JWT issue/verify.

Deferred (documented in ROADMAP/ISSUES): PostgreSQL backend, LDAP/AD, token
refresh rotation, scheduled PDF reports.

### Added — Phase 4c (backup, restore & bulk configuration)
- `GET /api/v1/devices/:id/config/export` — download a device config snapshot (JSON).
- `POST /api/v1/devices/:id/config/import` — restore protocols/syslog/routes from a
  snapshot (network optional, reboots); each section audited.
- `GET /api/v1/backup` — consistent SQLite database snapshot via `VACUUM INTO`,
  safe on a live DB; in-place restore is manual (documented).
- `POST /api/v1/bulk/config` — apply syslog or protocols to many devices
  concurrently, audited per device.
- Frontend: Export/Restore buttons on the Configuration tab (restore gated by a
  confirm dialog with an opt-in network checkbox); real Database export in
  Settings → Backup & Restore; new Settings → Bulk Configuration tab.

### Added — Phase 4b (configuration writes & device actions)
- Write endpoints proxying validated PUTs to the device, each recorded in a new
  audit log (`audit_events` table):
  - `PUT /api/v1/devices/:id/config/network` (reboots device), `/config/protocols`,
    `/config/syslog`, `/config/routes`
  - `POST /api/v1/devices/:id/reboot`, `/config-reset`
  - `GET /api/v1/audit` — config-change / action history
- Server-side validation: IPv4 for network/syslog/route gateways, CIDR for route
  destinations, port ranges, and config-reset scope allow-list.
- Editable **Configuration** tab: per-section Edit toggles with forms (network,
  protocols, syslog, static routes) and a **Device Actions** card (reboot, four
  config-reset scopes). Every device-affecting write is gated behind a
  confirmation dialog; network changes and resets are flagged as reboot/danger.
- Configuration **audit log** shown in the device Logs tab.
- Tests for IPv4 validation helpers.

### Added — Phase 4a (read-only configuration)
- `GET /api/v1/devices/:id/config` — on-demand, **read-only** aggregation of the
  device's configuration: network (`/self/ipconfig`), system (`/self/system`),
  protocols (`/self/protocols`), syslog (`/self/syslog`), static routes
  (`/self/static_route`), and DNS (`/self/diag/dns`). GET-only — no device writes.
- New **Configuration** tab on the device detail page rendering all sections
  read-only, fetched on demand (no background refetch), with a manual refresh
  and a "read-only / editing later" banner.
- `EmsfpClient.FetchConfig` (best-effort per endpoint) + `asString` helper, with
  a unit test.

## [0.3.0] — 2026-06-16

### Added

**Alerting & notifications**
- Configurable health thresholds via a new `alerting` config section:
  `temp_warning_c`, `temp_critical_c`, `response_warning_ms` — applied in the
  status engine (response-time and warning-temperature now raise warnings).
- Status-transition detection: every `online↔warning↔critical↔offline` change is
  recorded as an `AlertEvent` (new table) with from/to status and a message.
- Outbound webhook notifications on transitions into configured states
  (`alerting.webhook_on`, default `critical`/`offline`). Payload is
  Slack-compatible (`text`) and carries the structured event for generic consumers.
- `GET /api/v1/alerts` — status-transition history (optional `device` filter).
- `GET /api/v1/config` — effective non-sensitive runtime config (webhook URL
  surfaced only as enabled/disabled).
- Alert history is pruned alongside poll history by the daily retention job.

**Trends & export**
- Device-card temperature **sparklines** (pure SVG — kept out of the recharts bundle).
- PTP-offset already trended in Phase 2; **CSV export** of poll history via
  `GET /api/v1/devices/:id/history.csv` and an Export button on the Monitoring tab.

**Frontend**
- Logs tab now shows the device's **status-change history** beneath active alarms.
- New Settings → **Alerting** tab displaying the effective thresholds and webhook
  state; settings routing generalised to `/settings/:tab`.

**Tests**
- Webhook notifier tests (Slack-compatible payload, gating) and expanded
  status-derivation tests for the configurable thresholds.

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
