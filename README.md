# Embrionix Dashboard

A production-quality monitoring and management platform for **Embrionix EM6** devices, built with Go and React.

![Phase](https://img.shields.io/badge/Phase-5%20Enterprise-blue)
![Go](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go)
![React](https://img.shields.io/badge/React-18-61DAFB?logo=react)
![License](https://img.shields.io/badge/License-MIT-green)

---

## Features

- **Device Inventory** — Add, edit, remove, and search EM6 devices with dual-IP (Red/Blue) support
- **Live Dashboard** — Card and table views with real-time status, a fleet-wide alarm panel, and an auto-refresh countdown
- **Comprehensive Polling** — Every EM6 health/telemetry endpoint is collected: device info, system health, detailed PTP/refclk, firmware banks, licenses, control-plane ethernet counters, per-interface (e1/e2) config, LLDP neighbours, SFP DDM, media-flow telemetry, and SDI — see [API.md → EM6 endpoint coverage](API.md#em6-endpoint-coverage)
- **Dual-Path Reachability** — Independent L4 probe of the Red and Blue management paths each cycle
- **Historical Metrics** — SQLite time-series for temperature, fan, SFP TX/RX power, PTP offset, and response time, with device-card sparklines, CSV export, and a daily pruning job
- **Alerting** — Configurable thresholds, a per-device status-change history, and Slack-compatible/generic webhook notifications on transitions into critical/offline
- **Per-Device Detail** — Overview (health, PTP, firmware), Interfaces (e1/e2, LLDP, ethernet, media flows), SFP Modules, Monitoring charts, and Logs (alarms + status history) tabs
- **Configuration Management** — Read/write device config (network, protocols, syslog, routes), reboot & reset actions behind confirmation dialogs, per-device snapshot export/restore, SQLite database backup, and bulk apply across devices — all audited
- **Authentication & RBAC** (optional, off by default) — Local accounts (bcrypt) + JWT, three roles (viewer/operator/admin), API-key access for integrations, and a Users admin screen
- **Operator UX** — Toast notifications, keyboard shortcut (press **N** to add a device), live API-connectivity indicator
- **Settings** — Polling, alerting, bulk configuration, backup & restore, and user management

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
The frontend dev server proxies `/api` and `/health` to the Go backend on port 8081.

### Build for production

The frontend is **embedded into the binary** (`go:embed`), so production is a
single self-contained, self-updatable executable that serves both the UI and API.

```bash
# 1. Build the frontend
cd web && npm run build && cd ..

# 2. Copy it into the embed directory
rm -rf internal/webui/dist && mkdir -p internal/webui/dist
cp -r web/dist/* internal/webui/dist/

# 3. Build the binary (inject the version)
go build -ldflags="-X github.com/embrionix/dashboard/internal/version.Version=v0.6.0" \
  -o embrionix-dashboard ./cmd/server/

# 4. Run — UI + API on one port
./embrionix-dashboard configs/config.yaml
```

Tagging a release (`git tag v0.6.0 && git push --tags`) runs the **Release**
workflow, which does all of the above for every platform and publishes raw
binaries + `checksums.txt`. A running instance checks GitHub Releases and offers
an in-app **Update** pop-up (admin self-update; see the `updates` config below).

---

## Configuration

Copy and edit `configs/config.yaml`:

```yaml
server:
  port: 8081

database:
  path: "./data/embrionix.db"

polling:
  interval_seconds: 30          # How often to poll each device
  timeout_seconds: 10           # Per-request timeout
  retry_count: 2
  icmp_enabled: true            # Independent L4 reachability probe per Red/Blue path
  history_retention_days: 30    # Prune poll/alert history older than this (0 = keep forever)

alerting:
  temp_warning_c: 70            # Core temp (°C) raising a warning
  temp_critical_c: 75           # Core temp (°C) raising a critical alarm
  response_warning_ms: 2000     # API response time (ms) raising a warning
  webhook_url: ""               # Slack-compatible/generic webhook; empty disables notifications
  webhook_on: [critical, offline]   # fire a webhook on transition INTO these states

reports:
  enabled: false                # scheduled webhook summary (the PDF is always available on demand)
  cron: "0 8 * * 1"             # 5-field cron — default Mondays 08:00; needs alerting.webhook_url

updates:
  enabled: true                 # check GitHub Releases and allow admin in-app self-update
  repo: "jojo024/Embrionix-Dashboard"  # GitHub owner/name to check
  check_interval_hours: 6       # how often to poll for new releases

auth:
  enabled: false                # OFF by default — no login. Set true to require authentication.
  jwt_secret: ""                # REQUIRED when enabled (long random string / EMB_AUTH_JWT_SECRET)
  admin_username: "admin"       # seeded on first start when enabled
  admin_password: ""            # blank → random password generated and logged once
  api_key: ""                   # optional X-API-Key for integrations (admin-equivalent)

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
| [PERFORMANCE.md](PERFORMANCE.md) | Network/device impact analysis & tuning |
| [INSTALLATION.md](INSTALLATION.md) | Detailed install guide |
| [CONTRIBUTING.md](CONTRIBUTING.md) | How to contribute |
| [CHANGELOG.md](CHANGELOG.md) | Version history |
| [ISSUES.md](ISSUES.md) | Known limitations / unsupported API features |

---

## License

MIT
