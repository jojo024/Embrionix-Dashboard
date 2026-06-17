# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Quick Reference

### Development Commands

**Backend:**
```bash
# Run the server (watches configs/config.yaml)
go run ./cmd/server/

# Run with custom config
go run ./cmd/server/ ./configs/config.yaml

# Build checks
go vet ./...
go build ./...

# Linting (if golangci-lint is installed)
golangci-lint run ./...
```

**Frontend:**
```bash
# Navigate to web directory first
cd web

# Install dependencies
npm install

# Development server (proxies /api and /health to :8081)
npm run dev

# Build for production (output: web/dist/)
npm run build

# Type-check (no emit, used in CI)
npx tsc --noEmit

# Linting
npm run lint
```

**Production Build:**
```bash
# Full cross-platform build (see scripts/build.sh or scripts/build.ps1)
./scripts/build.sh v0.7.0

# Or on Windows:
.\scripts\build.ps1 -Version "0.7.0"

# Windows/Linux dev-only (no embedded UI):
CGO_ENABLED=0 go build ./cmd/server/
```

**Testing:**
- No dedicated test command (project status: Phase 5 complete, minimal test suite)
- Code review via `go vet ./...`, `go build ./...`, `npx tsc --noEmit`, `npm run build`

---

## Architecture Overview

### Backend Layers (Clean Architecture)
```
cmd/server/main.go
  ├── config.Load()          [internal/config/]
  ├── database.Open()        [pkg/database/]
  ├── logger.Init()          [pkg/logger/]
  │
  ├── Repositories           [internal/repositories/]
  │   ├── DeviceRepository   (GORM Device CRUD, queries)
  │   ├── PollRepository     (PollResult history + AppSettings)
  │   └── UserRepository     (auth: users, roles, API keys)
  │
  ├── Services               [internal/services/]
  │   ├── DeviceService      (device CRUD, dual-IP management)
  │   ├── PollingService     (goroutine scheduler, status cache, alerting)
  │   ├── EmsfpClient        (HTTP client for emSFP REST API on EM6 devices)
  │   ├── AlertService       (threshold logic, status transitions)
  │   ├── Notifier           (webhook delivery: Slack-compatible + generic)
  │   ├── UpdateService      (GitHub Release checker, self-update)
  │   └── AuthService        (JWT, password hashing, user/role management)
  │
  └── API Router             [internal/api/]
      ├── middleware/        (CORS, logger, recovery, auth JWT)
      └── handlers/          (devices, monitoring, health, settings, auth, etc)
```

**Data Flow — Device Poll Cycle:**
1. `PollingService.pollAll()` runs on a timer (every `polling.interval_seconds`)
2. For each device: `probeDualPath()` → TCP-connect Red + Blue (concurrent)
3. `EmsfpClient.Poll()` fetches ~15 endpoints from `/emsfp/node/v1`
   - Mandatory: `/self/information` (device type, firmware)
   - Best-effort: everything else (failures tolerated, don't fail poll)
4. `deriveStatus()` applies configured thresholds (temp warning/critical, response time)
5. Write `PollResult` to SQLite (time-series for temp, fan, SFP power, PTP offset)
6. Update in-memory `results` map (sync.RWMutex) — `GET /api/v1/devices` reads this, **zero DB queries for live status**
7. Compare new vs. old status; write `AlertEvent` if changed
8. Send webhook if transition is in `alerting.webhook_on` list

**Key In-Memory Cache:**
- `PollingService.results map[string]*pollState` — holds the latest poll per device
- Readers (HTTP handlers) use `sync.RWMutex` to read without DB access
- Only historical queries (charts, logs) hit the database

### Frontend Architecture
```
App (React 18 + TypeScript + Vite)
└── BrowserRouter + Suspense
    └── ToastProvider (transient feedback)
        └── Layout (sidebar + topbar with live API indicator)
            ├── Dashboard           (card/table view + FleetAlarmPanel + refresh countdown)
            ├── DevicesPage         (inventory CRUD, keyboard shortcut "N" to add)
            ├── DeviceDetail        (tabbed per-device view)
            │   ├── OverviewTab     (health, PTP, system, firmware)
            │   ├── InterfacesTab   (e1/e2, LLDP, ethernet, media flows, SFP)
            │   ├── SFPTab          (DDM per port)
            │   ├── MonitoringTab   (Recharts: temp, SFP power, PTP offset, response time)
            │   └── LogsTab         (alarms + status history)
            ├── MonitoringPage      (fleet-wide health table + bar chart)
            └── SettingsPage        (polling, alerting, device mgmt, backup/restore, users)
```

**State Management:**
- React Query handles all server state (caching, refetch, loading/error)
- No Redux/Zustand; component props + query keys sufficient
- Refetch intervals: 30s for device lists/summary/alarms; 60s for history
- Transient UI (toasts, loading) flows through small `ToastProvider` context

**Code-splitting:**
- Pages wrapped in `React.lazy()` + `Suspense` to keep entry bundle small
- Recharts-heavy pages (Monitoring, DeviceDetail) lazy-loaded
- Vendor libs (recharts, react, react-query) split via `manualChunks` in `vite.config.ts`

---

## Critical Constraints & Rules

### emSFP API Integration
1. **Never assume an endpoint exists** — All `/emsfp/node/v1/...` paths must be verified against `documentations/api_e+.html` before use
2. **No authentication required** — The `auth-usvc.json` docs are for NEP Broadcast Control, not device-level polling
3. **Best-effort polling** — Only `/self/information` is mandatory. All other endpoints fail silently; device stays online if that one succeeds
4. **Verified endpoints** — See [API.md → EM6 endpoint coverage](API.md#em6-endpoint-coverage) for the full mapping

### Go Codebase
1. **No CGO / Pure-Go SQLite** — Use `glebarez/sqlite` (wraps modernc.org/sqlite), **never** `mattn/go-sqlite3`
   - All builds must use `CGO_ENABLED=0` (Windows dev has no GCC; GitHub Actions also enforces this)
   - If a package requires CGO, find a pure-Go alternative or reject the dependency
2. **Code quality checks (must pass):**
   - `go vet ./...` 
   - `go build ./...`
3. **Exported symbols** — Must have doc comments if part of public API
4. **No panics in production paths** — Use `pkg/logger.Error()` and return errors instead

### Frontend Codebase
1. **Type safety** — `npx tsc --noEmit` must pass (strict mode)
2. **Build validation** — `npm run build` must produce a clean `web/dist/`
3. **API client centralization** — All server calls go through `src/api/client.ts`
4. **React Query hooks** — New server-state hooks go in `src/hooks/`
5. **Component design** — Small, single-purpose components; avoid shared component state

### Documentation Discipline
- Update `API.md` for any new endpoint
- Update `CHANGELOG.md` for user-visible changes
- Record unsupported features in `ISSUES.md` with rationale
- Keep `TODO.md` current for in-flight work

### Version & Release
- Single source of truth: `internal/version.Version` (injected via ldflags at build time)
- Release workflow: `git tag vX.Y.Z && git push --tags` triggers GitHub Actions → per-platform binaries + `checksums.txt`
- Self-update downloads matching binary, verifies SHA-256 vs checksums.txt, swaps in-place via `minio/selfupdate`, relaunches

---

## Key Files & Entry Points

| File | Purpose |
|------|---------|
| `cmd/server/main.go` | Application entry point; initialization order: config → logger → DB → repos → services → router |
| `internal/api/router.go` | Gin router setup, route registration |
| `internal/services/polling.go` | Scheduler, in-memory cache, status derivation, alerting |
| `internal/services/emsfp_client.go` | HTTP client for emSFP REST API (device polling) |
| `web/src/App.tsx` | React entry, router, provider setup |
| `web/src/api/client.ts` | Centralized HTTP client (React Query queries) |
| `configs/config.yaml` | Default configuration; all keys can be overridden with `EMB_` env vars |
| `ISSUES.md` | API limitations, deferred features, design decisions |
| `API.md` | REST endpoint reference (dashboard API) + emSFP endpoint coverage |
| `ARCHITECTURE.md` | System design, data flow, schema, decisions |

---

## Database Schema

| Table | Purpose |
|-------|---------|
| `devices` | Device inventory (name, IPs, enabled/disabled, monitoring flag) |
| `poll_results` | Time-series snapshots (device_id, temp, fan, SFP power, PTP offset, response_time) |
| `alert_events` | Status-transition history (device_id, old_status, new_status, timestamp) |
| `app_settings` | Key/value config store (not used for primary config; see `config.yaml`) |
| `users` | Auth: username, password_hash, role (viewer/operator/admin) |
| `api_keys` | Auth: API key + admin-equivalent role for integrations |

**SQLite specifics:**
- WAL mode enabled (concurrent reads, single-writer writes)
- MaxOpenConns=1 (single write connection via GORM)
- `glebarez/sqlite` is pure-Go (no CGO)

---

## Development Workflow

1. **Issue First** — Create a GitHub issue (bug/feature) before branching
2. **Branch Naming** — `feature/<slug>`, `fix/<slug>`, `docs/<slug>`, `chore/<slug>`
3. **Commit Style** — Imperative mood, present tense (e.g., "Add SFP DDM chart", "Fix polling panic when device has no IP")
4. **PR Description** — Reference the issue; summarize what changed and why
5. **Code Review Checklist:**
   - Does it touch device communication? Verified against `api_e+.html`?
   - Happy path tested? Error handling graceful (no panics)?
   - Docs updated (API.md, CHANGELOG.md, ISSUES.md if needed)?
   - `go vet ./...`, `go build ./...`, `npx tsc --noEmit`, `npm run build` pass?
6. **Merge to main** — After review + CI passes

---

## Common Tasks

### Adding a New emSFP Polling Metric
1. Verify endpoint in `documentations/api_e+.html`
2. Add field to `models.DevicePollingData` (Go struct)
3. Fetch in `EmsfpClient.Poll()` (best-effort)
4. Map to `PollResult` in `PollingService.pollDevice()` if time-series persistence needed
5. Add TypeScript type to `web/src/types/index.ts` (or inline interface)
6. Render in relevant UI tab (OverviewTab, SFPTab, MonitoringTab)
7. Update [API.md](API.md#em6-endpoint-coverage)

### Adding a New Dashboard API Endpoint
1. Create handler in `internal/api/handlers/`
2. Register route in `internal/api/router.go` (add middleware if auth needed)
3. Add service method in `internal/services/` if business logic required
4. Document in [API.md](API.md)

### Adding a New UI Page or Component
1. Create component file in `web/src/` (follow existing hierarchy)
2. If page-level (lazy-loaded): wrap in `React.lazy()`, register route in `App.tsx`, reference in `Suspense`
3. Use `src/api/client.ts` for server calls (React Query hooks)
4. Render in layout; test in dev server (`npm run dev` from web/)

### Debugging the Poll
1. Backend: Check logs via `pkg/logger` (Zap); `internal/services/polling.go` is the orchestrator
2. Frontend: Check `/api/v1/devices` response in DevTools → Network; compare to React Query cache state
3. In-memory cache: `PollingService.results` only, not persisted during poll interval
4. Database: `data/embrionix.db` (SQLite; use any SQLite browser or `sqlite3 CLI)

---

## Performance & Constraints

### Device/Network Impact (Minimal-Impact Polling)
- **Tiered fetch** — Dynamic health (~6 GETs) every cycle; static/heavy only every `polling.full_every` cycles (default 10)
- **HTTP keep-alive** — One TCP connection per device per poll (reused across requests)
- **Bounded concurrency** — `polling.max_concurrent_polls` (default 8) caps simultaneous device polls
- See [PERFORMANCE.md](PERFORMANCE.md) for rationale and load analysis

### In-Memory State
- Poll cache is **not persisted** between restarts (intentional; devices are live-monitored)
- History (for charts/logs) lives in SQLite; pruned by daily cron job (configured in alerting section)

---

## Configuration (configs/config.yaml)

Key sections:
- **server.port** — Default 8081; set `EMB_SERVER_PORT=9090` to override
- **polling.interval_seconds** — Poll frequency (default 30s)
- **polling.icmp_enabled** — TCP reachability probe for dual-path (Red/Blue)
- **alerting.temp_warning_c / temp_critical_c** — Thresholds for status
- **alerting.webhook_url** — Slack-compatible webhook (empty = disabled)
- **auth.enabled** — OFF by default; set true + jwt_secret to require login
- **updates.enabled / repo** — GitHub Release checker for self-updates

All values overridable via `EMB_<UPPERCASE_PATH>` environment variables (e.g., `EMB_POLLING_INTERVAL_SECONDS=60`).
