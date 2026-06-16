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

Create a new device.

**Request body**
```json
{
  "name": "EM6-MCR-01",
  "description": "Main control room encoder",
  "location": "MCR",
  "rack": "Rack 3, Unit 12",
  "serial_number": "EMB-2024-00001",
  "model": "Embox6",
  "firmware_version": "",
  "management_ip_red": "192.168.1.100",
  "management_ip_blue": "192.168.2.100",
  "tags": "production,encoding",
  "notes": "",
  "monitoring_enabled": true
}
```

**Response 201** — Created device with generated `id`.

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
L4 (TCP-connect) probe against **both** the Red and Blue management IPs, in
addition to the full REST poll. The results populate `reachable_red` and
`reachable_blue` on the device and each `PollResult`.

> **Why TCP, not ICMP?** Raw ICMP echo sockets require elevated privileges on
> Windows (and `CAP_NET_RAW` on Linux). To keep the dashboard runnable as an
> unprivileged process, reachability uses a TCP connect to the management port,
> which exercises the same L3 path without raw sockets. This trade-off is
> recorded in [ISSUES.md](ISSUES.md).

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
