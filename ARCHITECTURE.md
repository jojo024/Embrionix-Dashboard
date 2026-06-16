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
App (BrowserRouter)
└── Layout (sidebar + topbar)
    ├── Dashboard         — card/table view of all devices
    ├── DevicesPage       — inventory CRUD
    ├── DeviceDetail      — tabbed per-device view
    │     ├── OverviewTab
    │     ├── InterfacesTab
    │     ├── SFPTab
    │     ├── MonitoringTab  (Recharts line charts + poll table)
    │     └── LogsTab
    ├── MonitoringPage    — fleet-wide health table + bar chart
    └── SettingsPage      — device mgmt, polling config, backup, about
```

### State Management

React Query handles all server state (caching, background refetch, loading/error states). No global client-side store is needed — component props and query keys are sufficient. Refetch interval is 30 s for device lists, 60 s for history.

### Styling

Tailwind CSS v3 with a custom `surface` and `brand` palette designed for dark-mode NOC environments. Component variants (`.card`, `.btn-primary`, `.badge-online`, etc.) are defined in `@layer components` inside `index.css` to keep JSX clean.

---

## Data Flow — Device Poll Cycle

```
PollingService.pollAll()          (every N seconds)
  └─ for each monitoring-enabled device
       └─ EmsfpClient.Poll(ctx)
            ├─ GET /self/information   → firmware, type
            ├─ GET /self/ipconfig      → hostname, IP, MAC
            ├─ GET /self/system        → temp, fan, uptime
            ├─ GET /telemetry/node     → health + PTP
            ├─ GET /telemetry/ports    → per-port SFP summary
            └─ GET /port/{id}          → full DDM per port
       └─ Determine DeviceStatus (online / warning / critical / offline)
       └─ Write PollResult to SQLite
       └─ Update in-memory results map
```

Frontend GET `/api/v1/devices` → handler reads from in-memory map (no DB read for status), returns enriched Device array.

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
