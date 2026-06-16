# Installation Guide

## Requirements

| Component | Minimum |
|-----------|---------|
| Go | 1.24 |
| Node.js | 22 |
| npm | 11 |
| OS | Windows 10+, Ubuntu 22.04+, macOS 13+ |
| RAM | 256 MB |
| Disk | 100 MB |

Network access to Embrionix EM6 device management interfaces is required from the machine running the dashboard.

---

## Option 1 — Pre-built Binary (Recommended)

1. Download the latest release archive for your platform from the [Releases](../../releases) page.

2. Extract:
   ```bash
   tar -xzf embrionix-dashboard-linux-amd64.tar.gz
   cd release/
   ```

3. Edit `config.yaml` if needed (see [Configuration](#configuration)).

4. Run:
   ```bash
   ./embrionix-dashboard-linux-amd64
   ```

5. Open [http://localhost:8080](http://localhost:8080).

---

## Option 2 — Docker

```bash
docker pull ghcr.io/embrionix/dashboard:latest

docker run -d \
  --name embrionix-dashboard \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/configs/config.yaml:/app/configs/config.yaml \
  ghcr.io/embrionix/dashboard:latest
```

### Docker Compose

```bash
git clone https://github.com/embrionix/dashboard.git
cd dashboard
docker-compose up -d
```

---

## Option 3 — Build from Source

### 1. Clone the repository

```bash
git clone https://github.com/embrionix/dashboard.git
cd dashboard
```

### 2. Build the frontend

```bash
cd web
npm install
npm run build
cd ..
```

### 3. Build the backend

```bash
go build -o embrionix-dashboard ./cmd/server/
```

### 4. Run

```bash
./embrionix-dashboard configs/config.yaml
```

---

## Configuration

Default config file: `configs/config.yaml`

```yaml
server:
  host: "0.0.0.0"   # Bind address
  port: 8080
  mode: "release"    # release | debug

database:
  path: "./data/embrionix.db"

logging:
  level: "info"              # debug | info | warn | error
  file: "./logs/embrionix.log"

polling:
  interval_seconds: 30
  timeout_seconds: 10
  retry_count: 2

cors:
  allowed_origins:
    - "http://localhost:5173"   # Dev only — remove in production
```

**Environment variable overrides** (prefix `EMB_`, dots become underscores):

```bash
EMB_SERVER_PORT=9090
EMB_POLLING_INTERVAL_SECONDS=60
EMB_DATABASE_PATH=/var/lib/embrionix/embrionix.db
```

---

## Windows Service (Optional)

To run the dashboard as a Windows service, use [NSSM](https://nssm.cc/):

```powershell
nssm install EmbrionixDashboard "C:\path\to\embrionix-dashboard.exe"
nssm set EmbrionixDashboard AppParameters "C:\path\to\configs\config.yaml"
nssm set EmbrionixDashboard AppDirectory "C:\path\to\"
nssm start EmbrionixDashboard
```

---

## Linux systemd (Optional)

Create `/etc/systemd/system/embrionix-dashboard.service`:

```ini
[Unit]
Description=Embrionix Dashboard
After=network.target

[Service]
Type=simple
User=embrionix
WorkingDirectory=/opt/embrionix-dashboard
ExecStart=/opt/embrionix-dashboard/embrionix-dashboard configs/config.yaml
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload
systemctl enable --now embrionix-dashboard
```

---

## Upgrading

1. Stop the running instance.
2. Replace the binary (and `web/` directory if using from source).
3. The database auto-migrates on startup — no manual schema changes required.
4. Restart.
