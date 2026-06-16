# Changelog

All notable changes to Embrionix Dashboard are documented here.  
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Unreleased]

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
