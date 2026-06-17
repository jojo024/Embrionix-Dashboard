# REST API Reference

Base URL: `http://localhost:8081` (configurable via `server.port`)

All responses are JSON. All request bodies must include `Content-Type: application/json`.

---

## Health

### `GET /health`

Returns server health status.

**Response 200**
```json
{
  "status": "ok",
  "uptime": "2h35m12s",
  "timestamp": "2026-06-16T14:22:00Z",
  "go_version": "go1.26.4",
  "memory_mb": 18
}
```

---

## Devices

### `GET /api/v1/devices`

Returns all devices enriched with live polling data.

**Response 200**
```json
{
  "devices": [
    {
      "id": "uuid",
      "name": "EM6-MCR-01",
      "management_ip_red": "192.168.1.100",
      "management_ip_blue": "192.168.2.100",
      "status": "online",
      "last_polled_at": "2026-06-16T14:21:55Z",
      "reachable_red": true,
      "reachable_blue": false,
      "polling_data": { ... }
    }
  ],
  "total": 1
}
```

`status` is one of: `online` | `warning` | `critical` | `offline` | `unknown`

`reachable_red` / `reachable_blue` reflect the independent L4 (TCP) probe of each
management path when `polling.icmp_enabled` is set (see
[Dual-path reachability](#dual-path-reachability)).

The `polling_data` object is the full live snapshot collected from the EM6. See
[EM6 endpoint coverage](#em6-endpoint-coverage) for everything it contains
(PTP/refclk, firmware banks, licenses, ethernet counters, per-interface config,
LLDP neighbour, media-flow telemetry, SDI, SFP DDM).

### `GET /api/v1/devices/:id`

Returns a single device enriched with polling data.

**Response 404** if device does not exist.

### `POST /api/v1/devices`

Create a new device. **Required:** `name`, and at least one of `management_ip_red` or `management_ip_blue`. The firmware version is automatically fetched from the device.

**Request body**
```json
{
  "name": "EM6-MCR-01",
  "description": "Main control room encoder",
  "location": "MCR",
  "rack": "Rack 3, Unit 12",
  "serial_number": "EMB-2024-00001",
  "model": "Embox6",
  "management_ip_red": "192.168.1.100",
  "management_ip_blue": "192.168.2.100",
  "tags": "production,encoding",
  "notes": "",
  "monitoring_enabled": true
}
```

**Response 201** — Created device with generated `id`. If the device was reachable, `firmware_version` is auto-populated; if not, it's left blank and can be added later.

### `PUT /api/v1/devices/:id`

Full update of an existing device. Same body as POST.

**Response 200** | **404**

### `DELETE /api/v1/devices/:id`

Remove a device. Also removes all associated poll history.

**Response 204** | **404**

---

## Monitoring

### `GET /api/v1/summary`

Fleet-wide device counts by status.

**Response 200**
```json
{
  "total": 12,
  "online": 8,
  "offline": 2,
  "warning": 1,
  "critical": 1,
  "unknown": 0
}
```

### `GET /api/v1/alarms`

Every active alarm across the fleet, for the dashboard alarm panel. Each entry
is one alarm attributed to a device; unreachable devices contribute a single
`"Device unreachable"` entry.

**Response 200**
```json
{
  "alarms": [
    {
      "device_id": "uuid",
      "device_name": "EM6-MCR-01",
      "status": "warning",
      "message": "PTP not locked (coarse lock)",
      "polled_at": "2026-06-16T14:21:55Z"
    }
  ],
  "total": 1
}
```

### `GET /api/v1/alerts`

Status-transition history (the alert log). Optional `device` query param scopes
to one device; `limit` caps the count (default 100).

**Response 200**
```json
{
  "alerts": [
    {
      "id": 42,
      "device_id": "uuid",
      "device_name": "EM6-MCR-01",
      "from_status": "online",
      "to_status": "critical",
      "message": "Device is now critical",
      "created_at": "2026-06-16T14:21:55Z"
    }
  ],
  "total": 1
}
```

> `GET /api/v1/alarms` returns *currently active* conditions; `GET /api/v1/alerts`
> returns the *historical record* of status changes.

### `GET /api/v1/config`

Effective non-sensitive runtime configuration (polling + alerting). The webhook
URL is reported only as `webhook_enabled` (boolean), never echoed back.

**Response 200**
```json
{
  "polling": { "interval_seconds": 30, "timeout_seconds": 10, "icmp_enabled": true, "history_retention_days": 30 },
  "alerting": { "temp_warning_c": 70, "temp_critical_c": 75, "response_warning_ms": 2000, "webhook_enabled": false, "webhook_on": ["critical", "offline"] }
}
```

### `GET /api/v1/devices/:id/history`

Historical poll results for a device.

**Query params**
| Param | Default | Description |
|-------|---------|-------------|
| `limit` | `100` | Max number of results (newest first) |
| `since` | — | RFC3339 timestamp; return results after this time |

**Response 200** — Array of `PollResult`:
```json
[
  {
    "id": 1,
    "device_id": "uuid",
    "polled_at": "2026-06-16T14:21:55Z",
    "reachable": true,
    "response_ms": 42,
    "core_temp": 49.5,
    "fan_speed": 4210,
    "core_voltage": 700,
    "port0_tx_power": 922,
    "port0_rx_power": 718,
    "port0_temp": 33.9,
    "port1_tx_power": null,
    "port1_rx_power": null,
    "port1_temp": null,
    "ptp_locked": true,
    "ptp_offset": 1950,
    "reachable_red": true,
    "reachable_blue": false,
    "error_message": ""
  }
]
```

### `GET /api/v1/devices/:id/history.csv`

Streams the device's poll history as a CSV download (`Content-Disposition:
attachment`). Columns: `polled_at, reachable, response_ms, core_temp, fan_speed,
core_voltage, port0_tx_power, port0_rx_power, ptp_locked, ptp_offset`. Optional
`limit` query param (default 1000).

### `POST /api/v1/devices/:id/poll`

Trigger an immediate on-demand poll of a device.

**Response 200**
```json
{
  "reachable": true,
  "polling_data": { ... }
}
```

**Response 503** if device is unreachable.

### `GET /api/v1/devices/:id/config`

Fetches the device's **read-only** configuration on demand (GET-only — never
writes to the device). Aggregates `/self/ipconfig`, `/self/system`,
`/self/protocols`, `/self/syslog`, `/self/static_route`, and `/self/diag/dns`.
Each section is best-effort and omitted if the device type does not implement it.

**Response 200**
```json
{
  "network": { "mac_address": "40:a3:6b:a0:1f:a6", "ip_addr": "192.168.39.48", "subnet_mask": "255.255.255.0", "gateway": "192.168.39.1", "hostname": "emsfp-a0-1f-a6", "port": "80", "dhcp_enable": "1", "ctl_vlan_id": "0", "ctl_vlan_pcp": "0", "ctl_vlan_enable": "0" },
  "system": { "staging_mode": 0, "min_fan_speed": 32, "smpte_2022_7_class": "a" },
  "protocols": { "mdns_enable": "1", "ember_server_port": "3344", "sap_announcement_enable": "0" },
  "syslog": { "server": "192.168.3.111", "port": 514, "enable": true, "monitoring": { "common": { "ptp_event": true } } },
  "static_routes": [ { "name": "route_1", "destination": "192.168.39.0/24", "gateway": "192.168.4.1" } ],
  "dns": { "server_address": "0.0.0.0", "domain_name": "" }
}
```

**Response 503** if the device is unreachable.

## Authentication & users (Phase 5)

Auth is **disabled by default**; when off, all endpoints are open (implicit admin).
When `auth.enabled` is true, send `Authorization: Bearer <jwt>` (from login) or
`X-API-Key: <key>`. RBAC: viewer = GET reads, operator = + writes/actions,
admin = + user management. Under-privileged calls return **403**; missing/invalid
credentials return **401**.

### `POST /api/v1/auth/login`
Body `{ "username": "...", "password": "..." }` → `{ "token": "...", "user": { "id", "username", "role" } }`.

### `GET /api/v1/auth/me`
Reports `{ "auth_enabled": bool, "username": "...", "role": "..." }` for the caller.

### `GET /api/v1/report.pdf`
Downloads a PDF fleet report (status counts, active alarms, recent status
changes). Viewer+. A text version of the same summary is delivered to the
alerting webhook on the `reports.cron` schedule when `reports.enabled` is true.

### `GET /api/v1/export/ansible`
Device inventory as Ansible dynamic-inventory JSON (group `emsfp`).

## Updates

### `GET /api/v1/version`
Running version and cached update status. Viewer+.
```json
{
  "current_version": "v0.6.0",
  "latest_version": "v0.7.0",
  "update_available": true,
  "release_url": "https://github.com/.../releases/tag/v0.7.0",
  "release_notes": "…",
  "checked_at": "2026-06-17T08:00:00Z",
  "enabled": true
}
```

### `POST /api/v1/update/check`
Forces an immediate re-check against GitHub Releases (operator+).

### `POST /api/v1/update`
Downloads the latest release binary for this platform, verifies its SHA-256
against `checksums.txt`, swaps the running binary and **restarts the server**
(admin only). Returns `{ "status": "updating" }`; the process relaunches and the
UI reloads when `/health` reports the new version. Requires `updates.enabled`.

### `GET/POST /api/v1/users`, `PUT/DELETE /api/v1/users/:id` (admin)
List / create / update (role or password) / delete users. The last remaining
account cannot be deleted.

## Configuration writes & device actions (Phase 4b)

All writes proxy a PUT to the device and are recorded in the audit log
(`GET /api/v1/audit`). Inputs are validated server-side (IPv4 / CIDR / port
range). A device-side failure returns **502** with the error and the audit entry.

### `PUT /api/v1/devices/:id/config/network`
Writes `/self/ipconfig`. **The device reboots to apply.** Body: `NetworkUpdate`
(`ip_addr`, `subnet_mask`, `gateway`, `hostname`, `port`, `dhcp_enable`,
`ctl_vlan_id`, `ctl_vlan_pcp`, `ctl_vlan_enable`). Static IP fields are required
unless `dhcp_enable` is `"1"`.

### `PUT /api/v1/devices/:id/config/protocols`
Writes `/self/protocols` (`mdns_enable`, `ember_server_port`, `sap_announcement_enable`).

### `PUT /api/v1/devices/:id/config/syslog`
Writes `/self/syslog`. Body: `{ "server": "192.168.3.10", "port": 514, "enable": true, "monitoring": {...} }`.

### `PUT /api/v1/devices/:id/config/routes`
Writes `/self/static_route`. Body: `{ "routes": [ { "destination": "192.168.5.0/24", "gateway": "192.168.1.1" } ] }` (max 5).

### `POST /api/v1/devices/:id/reboot`
Reboots the device (`/self/system` `reboot=1`).

### `POST /api/v1/devices/:id/config-reset`
Resets configuration (`/self/system`). Body: `{ "scope": "flows" | "application" | "generic" | "system" }`. **Irreversible; the device reboots.**

### `GET /api/v1/audit`
Audit log of config writes / actions, newest first. Optional `device` and `limit`.

**Response 200**
```json
{ "events": [ { "id": 7, "device_id": "uuid", "device_name": "EM6-MCR-01", "action": "config.network", "detail": "network: dhcp=0 ip=192.168.1.50 ...", "success": true, "message": "", "created_at": "2026-06-16T15:00:00Z" } ], "total": 1 }
```

### `GET /api/v1/devices/:id/reachability`

Check reachability of Red and Blue management interfaces independently.

**Response 200**
```json
{
  "red":  { "ip": "192.168.1.100", "reachable": true,  "response_ms": 12 },
  "blue": { "ip": "192.168.2.100", "reachable": false, "response_ms": 10002 }
}
```

---

## Settings

### `GET /api/v1/settings/:key`

Retrieve a single application setting.

**Response 200**
```json
{ "key": "polling.interval_seconds", "value": "30" }
```

**Response 404** if key does not exist.

### `PUT /api/v1/settings/:key`

Set an application setting.

**Request body**
```json
{ "value": "60" }
```

**Response 200**

---

## Dual-path reachability

When `polling.icmp_enabled` is `true`, every poll cycle runs an independent
reachability probe against **both** the Red and Blue management IPs, in addition
to the full REST poll. The results populate `reachable_red` and `reachable_blue`
on the device and each `PollResult`.

- **Red** is always probed with a **TCP connect** to port 80 — it runs the HTTP
  management API, so a successful connect proves the device is actually serving.
- **Blue** is probed per `polling.blue_probe`: **`icmp`** (default) or **`tcp`**.
  EM6 second/Blue interfaces typically answer ICMP but **do not** run the HTTP
  management server, so a TCP probe there would falsely read offline. ICMP uses
  the operating system's `ping` command (no raw sockets, no elevated privileges).

> **Why the OS `ping` and not a raw socket?** Raw ICMP echo sockets require
> elevated privileges on Windows (and `CAP_NET_RAW` on Linux). Shelling out to the
> system `ping` exercises the same L3 path while keeping the dashboard an
> unprivileged process. See [ISSUES.md](ISSUES.md).

---

## EM6 endpoint coverage

The polling engine ([`emsfp_client.go`](internal/services/emsfp_client.go))
collects health and telemetry from the EM6 REST API (base
`http://<ip>/emsfp/node/v1`). The table below maps every documented endpoint to
its monitoring status. "Polled" endpoints feed `polling_data`; "config-plane"
endpoints describe media routing/configuration and are intentionally **not**
polled in the monitoring product (rationale in [ISSUES.md](ISSUES.md)).

| emSFP endpoint | Polled | Surfaced as |
|----------------|:------:|-------------|
| `/self/information` | ✅ | Versions, device type, platform HW |
| `/self/ipconfig` | ✅ | Hostname, IP, MAC, DHCP |
| `/self/system` | ✅ | Core temp, fan, voltage, uptime |
| `/self/firmware` | ✅ | Firmware banks (slot/version/active/default) |
| `/self/license` | ✅ | Licensed feature map |
| `/self/diag/refclk` | ✅ | PTP lock status, offset, mean delay, counters |
| `/self/diag/ethernet` | ✅ | Control-plane TX/RX packets + RX errors |
| `/self/diag/common` | ✅ | Video bandwidth usage, watchdog, IPv4 drops |
| `/self/interfaces` | ✅ | Per-interface (e1/e2) IP, gateway, DHCP, VLAN |
| `/lldp` | ✅ | Discovered neighbour (chassis, port, TTL) |
| `/telemetry/node` | ✅ | Health + refclk summary |
| `/telemetry/ports` | ✅ | Per-port SFP TX/RX power, temperature |
| `/telemetry/devices` | ✅ | Media-flow packet counters, validity |
| `/port`, `/port/{id}` | ✅ | SFP DDM (temp, VCC, bias, TX/RX power, alarms) |
| `/sdi` | ✅ | SDI operating bit rate |
| `/self/protocols`, `/self/syslog`, `/self/static_route`, `/self/diag/dns` | ❌ | Config-plane (settings, not health) |
| `/sources`, `/receivers`, `/senders`, `/flows`, `/sdp`, `/route`, `/clean_switch` | ❌ | Media routing — Phase 4 config management |
| `/black_burst`, `/sdi_input`, `/sdi_audio`, `/sdi_output` | ❌ | Media I/O config — Phase 4 |

All polled endpoints are fetched best-effort: a failure on any single endpoint
is tolerated (the device may not implement it for its type) and does not abort
the poll. Only `/self/information` is mandatory for a device to count as
reachable.

See `documentations/api_e+.html` for the full emSFP API reference.
