# Architecture

## Overview

Embrionix Dashboard is a two-tier web application:

```
Browser (React SPA)  ←→  Go HTTP Server  ←→  SQLite DB
                                ↕
                    Embrionix EM6 Devices (emSFP REST API)
```

The Go server serves both the REST API and (in production) the static React bundle. The polling engine runs as background goroutines concurrent with the HTTP server.

---

## Backend

### Layer Diagram

```
cmd/server/main.go
      │
      ├── config.Load()          internal/config/
      ├── database.Open()        pkg/database/
      ├── logger.Init()          pkg/logger/
      │
      ├── Repositories           internal/repositories/
      │     ├── DeviceRepository   (GORM queries)
      │     └── PollRepository     (history + settings)
      │
      ├── Services               internal/services/
      │     ├── DeviceService      (CRUD business logic)
      │     ├── PollingService     (scheduler + state cache)
      │     └── EmsfpClient        (HTTP client for emSFP API)
      │
      └── API Router             internal/api/
            ├── Middleware         (CORS, logger, recovery)
            └── Handlers           (devices, monitoring, health, settings)
```

### Key Design Decisions

**Clean architecture layering** — Handlers depend on services; services depend on repositories; repositories depend on GORM. No layer skips another.

**Pure-Go SQLite** — `glebarez/sqlite` (wraps `modernc.org/sqlite`) eliminates the CGO dependency, making cross-compilation straightforward with `CGO_ENABLED=0`.

**In-memory state cache** — `PollingService` keeps the latest poll result in a `sync.RWMutex`-protected map, so `/api/v1/devices` returns enriched data with zero database reads per request for live status.

**Concurrent polling** — Each device is polled in its own goroutine via `go func()`. The scheduler goroutine uses a `time.Ticker`; a `stop` channel enables clean shutdown.

**Configuration via Viper** — `configs/config.yaml` is the default source. Any key can be overridden with `EMB_`-prefixed environment variables, enabling Docker/Kubernetes deployments without file-mounting.

---

## Frontend

### Component Hierarchy

```
App (BrowserRouter, Suspense)
└── ToastProvider
    └── Layout (sidebar + topbar with live API indicator)
        ├── Dashboard         — card/table view + FleetAlarmPanel + RefreshCountdown
        ├── DevicesPage       — inventory CRUD (toasts, "N" shortcut)   [lazy]
        ├── DeviceDetail      — tabbed per-device view                  [lazy]
        │     ├── OverviewTab    (health, PTP, system, firmware)
        │     ├── InterfacesTab  (e1/e2, LLDP, ethernet, media flows, SFP)
        │     ├── SFPTab
        │     ├── MonitoringTab  (Recharts: temp, SFP power, PTP offset, response)
        │     └── LogsTab
        ├── MonitoringPage    — fleet-wide health table + bar chart     [lazy]
        └── SettingsPage      — device mgmt, polling config, about      [lazy]
```

Routes marked `[lazy]` are `React.lazy` + `Suspense` code-split so the
recharts-heavy pages stay out of the initial bundle. Vendor libraries (recharts,
React, React Query) are further split via `manualChunks` in `vite.config.ts`.

### State Management

React Query handles all server state (caching, background refetch, loading/error states). No global client-side store is needed — component props and query keys are sufficient. Refetch interval is 30 s for device lists, summary and alarms; 60 s for history. Transient UI feedback (CRUD/poll results) flows through a small `ToastProvider` context.

### Styling

Tailwind CSS v3 with a custom `surface` and `brand` palette designed for dark-mode NOC environments. Component variants (`.card`, `.btn-primary`, `.badge-online`, etc.) are defined in `@layer components` inside `index.css` to keep JSX clean.

---

## Data Flow — Device Poll Cycle

```
PollingService.pollAll()          (every N seconds)
  └─ for each monitoring-enabled device  (own goroutine)
       ├─ probeDualPath()        → TCP-connect Red + Blue (concurrent) → reachable_red/blue
       └─ EmsfpClient.Poll(ctx)  (against the reachable path)
            ├─ GET /self/information      → versions, type            (mandatory)
            ├─ GET /self/ipconfig         → hostname, IP, MAC
            ├─ GET /self/system           → temp, fan, voltage, uptime
            ├─ GET /telemetry/node        → health + PTP summary
            ├─ GET /telemetry/ports       → per-port SFP summary
            ├─ GET /self/diag/refclk      → detailed PTP (offset, delay, counters)
            ├─ GET /self/firmware         → firmware banks
            ├─ GET /self/license          → licensed features
            ├─ GET /self/diag/ethernet    → control-plane packet counters
            ├─ GET /self/diag/common      → video bandwidth, watchdog, drops
            ├─ GET /self/interfaces       → per-interface (e1/e2) config
            ├─ GET /lldp                  → neighbour
            ├─ GET /telemetry/devices     → media-flow packet counters
            ├─ GET /sdi                   → SDI bit rate
            └─ GET /port/{id}             → full DDM per port
       └─ deriveStatus()         → online / warning / critical / offline
       └─ Write PollResult to SQLite (temp, fan, SFP power, PTP offset, dual-path)
       └─ Update in-memory results map
```

Every endpoint except `/self/information` is best-effort: a single failure is
tolerated so device-type-specific endpoints (e.g. `/sdi`) don't fail the poll.
The full endpoint→data mapping is in [API.md](API.md#em6-endpoint-coverage).

Frontend GET `/api/v1/devices` → handler reads from in-memory map (no DB read for status), returns enriched Device array.

### Background jobs

Alongside the poll ticker, `PollingService` runs two more goroutines, all stopped
via the shared `stop` channel:

- **Poll scheduler** — `time.Ticker` at `polling.interval_seconds`.
- **History pruner** — daily `time.Ticker`; deletes `poll_results` older than
  `polling.history_retention_days` (skipped when `0`).

### Reachability probe

Dual-path reachability uses a **TCP connect** to the management port rather than
raw ICMP, so the server runs unprivileged. Gated by `polling.icmp_enabled`. See
[ISSUES.md](ISSUES.md) for the rationale.

---

## Database Schema

| Table | Purpose |
|-------|---------|
| `devices` | Device inventory (static config) |
| `poll_results` | Time-series health snapshots |
| `app_settings` | Key/value application configuration |

SQLite WAL mode is enabled for concurrent reads. A single write connection is used (`MaxOpenConns=1`).

---

## Security Considerations

- No authentication in Phase 1 (internal network tool assumption)
- CORS configured to restrict allowed origins
- SQL injection prevented by GORM parameterised queries
- No secrets stored; device IPs are not sensitive in this context
- Phase 5 will add RBAC and audit logging

---

## Extending

**Add a new API endpoint:**
1. Create handler in `internal/api/handlers/`
2. Register route in `internal/api/router.go`
3. Add service method in `internal/services/` if business logic is needed

**Add a new emSFP metric:**
1. Add field to `models.DevicePollingData`
2. Fetch in `EmsfpClient.Poll()`
3. Map to `PollResult` in `PollingService.pollDevice()` for persistence
4. Add to TypeScript `DevicePollingData` type
5. Render in the relevant UI tab
