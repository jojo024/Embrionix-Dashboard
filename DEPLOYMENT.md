# Deployment Guide

How to deploy Embrionix Dashboard on a live system. The app ships as a **single
self-contained binary** that serves both the web UI and the REST API and embeds
the frontend — there is nothing else to install (no Node, no separate web server,
no database engine; SQLite is built in).

> **Live broadcast system note.** This dashboard only ever *reads* from devices on
> a schedule and *writes* to them on explicit, audited operator actions. It is a
> "polite" poller (see [PERFORMANCE.md](PERFORMANCE.md)). Still, treat the first
> rollout cautiously: deploy alongside your existing monitoring, watch it for a
> day, then rely on it.

---

## 1. Prerequisites

- A host (Windows or Linux) on the **management network** that can reach each
  device's Red/Blue management IP on **TCP/80** (the emSFP REST API).
- One TCP port for the dashboard itself (default **8081**) reachable by the
  operators' browsers.
- Outbound HTTPS to `api.github.com` / `github.com` **only if** you want in-app
  update checking + self-update (otherwise set `updates.enabled: false`).

No database server, runtime, or web server is required.

---

## 2. Install

### Option A — download a release (recommended)

1. On the [Releases page](https://github.com/jojo024/Embrionix-Dashboard/releases),
   download the asset for your platform, e.g.
   `embrionix-dashboard-windows-amd64.exe` (+ its `.sha256`).
2. Verify the checksum:
   - Windows: `Get-FileHash .\embrionix-dashboard-windows-amd64.exe -Algorithm SHA256`
   - Linux: `sha256sum -c embrionix-dashboard-linux-amd64.sha256`
3. Place it in an install directory (e.g. `C:\Embrionix\` or `/opt/embrionix/`)
   alongside a `config.yaml`.

### Option B — build from source

See [README.md](README.md#build-for-production). The frontend is embedded via
`go:embed`, so a production build is: build the UI → copy into
`internal/webui/dist` → `go build` with the version ldflag.

---

## 3. Configure

Copy [`configs/config.yaml`](configs/config.yaml) next to the binary and edit it.
The settings that matter most for a production deployment:

```yaml
server:
  host: "0.0.0.0"      # bind all interfaces; use a specific IP to restrict
  port: 8081
  mode: "release"      # IMPORTANT: switch from "debug" to "release" in production

database:
  path: "./data/embrionix.db"   # persisted; back this up (see §7)

polling:
  interval_seconds: 30          # raise for large/slow fleets (see PERFORMANCE.md)
  full_every: 10
  max_concurrent_polls: 8

alerting:
  webhook_url: ""               # set a Slack/generic webhook to get notifications
  webhook_on: [critical, offline]

updates:
  enabled: true
  repo: "jojo024/Embrionix-Dashboard"
  restart_mode: "exit"          # IMPORTANT when running as a service (see §6)

auth:
  enabled: false                # see §8 to require login + RBAC
```

Any setting can also be supplied via environment variable with the `EMB_` prefix
and `_` separators, e.g. `EMB_SERVER_PORT=9000`, `EMB_AUTH_JWT_SECRET=...`.

---

## 4. First run & smoke test

```bash
./embrionix-dashboard config.yaml
```

Then verify:

```bash
curl http://localhost:8081/health
# {"status":"ok","version":"v0.6.0",...}
```

Open `http://<host>:8081/` in a browser — you should see the dashboard. Add a
device (name + at least one management IP); the firmware version auto-populates
from the device.

---

## 5. Data & logs

| Path (relative to the binary) | Contents |
|---|---|
| `./data/embrionix.db` | SQLite DB — inventory, poll history, alerts, audit log |
| `./logs/embrionix.log` | Application log |

Both directories are created on first run. Point them elsewhere via
`database.path` and `logging.file`.

---

## 6. Run as a service (recommended)

Running under a service manager gives you start-on-boot and restart-on-crash. **It
also cooperates with in-app self-update — set `updates.restart_mode: "exit"`** so
that after an update the process exits and the service manager starts the new
binary (in `"self"` mode the app would relaunch itself and fight the supervisor
for the port).

### Linux (systemd)

Use the provided unit [`deploy/embrionix-dashboard.service`](deploy/embrionix-dashboard.service):

```bash
sudo useradd --system --no-create-home embrionix
sudo mkdir -p /opt/embrionix && sudo cp embrionix-dashboard config.yaml /opt/embrionix/
sudo chown -R embrionix:embrionix /opt/embrionix
sudo cp deploy/embrionix-dashboard.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now embrionix-dashboard
sudo systemctl status embrionix-dashboard
```

### Windows (service via NSSM)

Use the provided script [`deploy/install-windows-service.ps1`](deploy/install-windows-service.ps1)
from an elevated PowerShell (requires [NSSM](https://nssm.cc)):

```powershell
.\deploy\install-windows-service.ps1 -InstallDir C:\Embrionix
```

This installs an auto-start service that restarts on exit (so self-update works).

---

## 7. Backups

- **Database:** download a consistent snapshot any time from **Settings → Backup &
  Restore → Export Database** (uses SQLite `VACUUM INTO`, safe while running), or
  `GET /api/v1/backup`. Schedule it via cron/Task Scheduler if you want automated
  copies. To restore: stop the service, replace `data/embrionix.db`, start again.
- **Config:** keep `config.yaml` in version control / your config management.
- **History retention** is bounded by `polling.history_retention_days` (default 30).

---

## 8. Security

- **Bind address.** `server.host: "0.0.0.0"` exposes the dashboard on all
  interfaces. Restrict to a management IP, or front it with a reverse proxy.
- **Authentication is off by default** (the app is open). To require login + RBAC:
  ```yaml
  auth:
    enabled: true
    jwt_secret: "<long random string>"   # or EMB_AUTH_JWT_SECRET
    admin_username: "admin"
    admin_password: ""                    # blank → a random one is generated & logged once
  ```
  Roles: **viewer** (read), **operator** (+ writes/device actions), **admin**
  (+ user management **and self-update**). The first start seeds the admin account.
- **TLS.** The server speaks plain HTTP. For encryption, front it with a reverse
  proxy (nginx/Caddy/IIS) terminating TLS and proxying to `127.0.0.1:8081`.
- **Device API** has no authentication (it's the emSFP design); keep the
  management network isolated as you already do for broadcast infrastructure.

---

## 9. Updates & rollback

### In-app self-update (admin)

When `updates.enabled` is true, a running instance checks GitHub Releases. When a
newer version exists, an **Update available** pop-up appears with **Update** /
**Dismiss**. An **admin** clicking Update downloads the matching binary, verifies
its SHA-256 against the release's `checksums.txt`, swaps the running binary, and
restarts (see `restart_mode`, §6). The page reloads automatically once the new
version is up (~10s of monitoring downtime).

> Validate the first real update (e.g. `v0.6.0 → v0.6.1`) on a **non-production**
> host before relying on it on the live box.

### Publishing a release

Cut a tag and push it — CI builds every platform, embeds the UI, injects the
version, and publishes raw binaries + `checksums.txt`:

```bash
git tag v0.6.0
git push origin v0.6.0
```

### Manual update / rollback

- **Manual update:** stop the service, replace the binary, start it.
- **Rollback:** the self-update leaves the previous binary as `<name>.old` next to
  the running one — stop the service, swap it back, start. Or just re-deploy the
  previous release asset. Application data (DB) is untouched by updates.

---

## 10. Troubleshooting

| Symptom | Check |
|---|---|
| UI loads but no devices update | Host can reach device mgmt IPs on TCP/80? Firewall? |
| Device shows "slow" badge | Normal for these devices if persistently >6s; see PERFORMANCE.md tuning |
| "frontend not built into this binary" | You're running a backend-only build; use a release asset or a full prod build (§2B) |
| Port already in use on start | Another instance running, or a self-update relaunch overlap (it retries binding for ~15s) |
| Update pop-up never appears | `updates.enabled`? Host can reach GitHub? A release tag exists? Current build is a real `vX.Y.Z` (a "dev" build never offers updates) |
| Self-update fails | Checked the log for checksum/download errors; ensure the release has the asset for this platform + `checksums.txt` |
| Update applied but didn't restart | Under a service manager? Set `updates.restart_mode: "exit"` (§6) |

---

## Quick reference

| | |
|---|---|
| Binary | single self-contained executable (UI + API + SQLite) |
| Default port | 8081 (`server.port`) |
| Data | `./data/embrionix.db` · Logs `./logs/embrionix.log` |
| Health | `GET /health` → `{status, version, uptime}` |
| Update check | `GET /api/v1/version` |
| Service (Linux) | [`deploy/embrionix-dashboard.service`](deploy/embrionix-dashboard.service) |
| Service (Windows) | [`deploy/install-windows-service.ps1`](deploy/install-windows-service.ps1) |
| Impact analysis | [PERFORMANCE.md](PERFORMANCE.md) |
| API reference | [API.md](API.md) |
