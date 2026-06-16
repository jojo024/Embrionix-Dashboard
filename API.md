# REST API Reference

Base URL: `http://localhost:8080`

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
      "polling_data": { ... }
    }
  ],
  "total": 1
}
```

`status` is one of: `online` | `warning` | `critical` | `offline` | `unknown`

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
    "error_message": ""
  }
]
```

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

## emSFP Device API (proxied — Phase 4)

The following emSFP endpoints will be proxied through the dashboard in Phase 4 to enable configuration management. They are **not currently exposed**.

| Method | emSFP Path | Purpose |
|--------|-----------|---------|
| GET/PUT | `/self/ipconfig` | Network settings |
| GET/PUT | `/self/system` | Reboot, config reset |
| GET/PUT | `/self/syslog` | Syslog server |
| GET | `/self/firmware` | Firmware slots |
| GET | `/telemetry/devices` | Flow telemetry |
| GET | `/lldp` | LLDP neighbours |

See `documentations/api_e+.html` for the full emSFP API reference.
