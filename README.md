# Embrionix Dashboard

A production-quality monitoring and management platform for **Embrionix EM6** devices, built with Go and React.

![Phase](https://img.shields.io/badge/Phase-1%20Foundation-blue)
![Go](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go)
![React](https://img.shields.io/badge/React-18-61DAFB?logo=react)
![License](https://img.shields.io/badge/License-MIT-green)

---

## Features (Phase 1)

- **Device Inventory** — Add, edit, remove, and search EM6 devices with dual-IP (Red/Blue) support
- **Live Dashboard** — Card and table views with real-time status, colour-coded by health
- **Background Polling** — Concurrent polling engine that pulls `self/information`, `self/system`, `telemetry/node`, `telemetry/ports`, and SFP DDM data
- **Historical Metrics** — SQLite time-series storage for temperature, fan speed, SFP TX/RX power, and response time
- **Per-Device Detail** — Overview, Interfaces, SFP Modules, Monitoring charts, and Logs tabs
- **Settings** — Polling configuration, device management, backup/restore groundwork

---

## Quick Start

### Prerequisites

| Tool | Version |
|------|---------|
| Go   | 1.24+   |
| Node.js | 22+ |
| npm  | 11+     |

### Run in development

```bash
# 1 — Backend
go run ./cmd/server/

# 2 — Frontend (separate terminal)
cd web && npm install && npm run dev
```

Open [http://localhost:5173](http://localhost:5173).  
The frontend dev server proxies `/api` and `/health` to the Go backend on port 8080.

### Build for production

```bash
# Build frontend into web/dist/
cd web && npm run build && cd ..

# Build backend binary
go build -o embrionix-dashboard ./cmd/server/

# Run
./embrionix-dashboard configs/config.yaml
```

---

## Configuration

Copy and edit `configs/config.yaml`:

```yaml
server:
  port: 8080

database:
  path: "./data/embrionix.db"

polling:
  interval_seconds: 30   # How often to poll each device
  timeout_seconds: 10    # Per-request timeout
  retry_count: 2

cors:
  allowed_origins:
    - "http://localhost:5173"
```

All values can be overridden with environment variables prefixed `EMB_`  
(e.g. `EMB_SERVER_PORT=9090`, `EMB_POLLING_INTERVAL_SECONDS=60`).

---

## Docker

```bash
# Build image
docker build -t embrionix-dashboard .

# Run with docker-compose
docker-compose up -d
```

---

## Project Structure

```
cmd/server/          — Application entry point
internal/
  api/               — Gin router, handlers, middleware
  models/            — GORM models (Device, PollResult, AppSetting)
  repositories/      — Database access layer
  services/          — Business logic + emSFP API client
  config/            — Configuration loader
pkg/
  database/          — SQLite setup
  logger/            — Zap logger wrapper
configs/             — Default config.yaml
web/                 — React/TypeScript frontend (Vite)
documentations/      — Vendor API reference (emSFP, auth-usvc)
.github/             — CI/CD workflows, issue/PR templates
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for the full design rationale.

---

## Documentation

| File | Contents |
|------|----------|
| [ARCHITECTURE.md](ARCHITECTURE.md) | System design and decisions |
| [API.md](API.md) | REST API reference |
| [ROADMAP.md](ROADMAP.md) | Phased feature plan |
| [INSTALLATION.md](INSTALLATION.md) | Detailed install guide |
| [CONTRIBUTING.md](CONTRIBUTING.md) | How to contribute |
| [CHANGELOG.md](CHANGELOG.md) | Version history |
| [ISSUES.md](ISSUES.md) | Known limitations / unsupported API features |

---

## License

MIT
