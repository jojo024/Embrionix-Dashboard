# Known Issues and API Limitations

This file documents features that could not be implemented as desired due to emSFP API constraints, the decisions taken, and proposed alternatives.

---

## Confirmed Supported (polled)

Verified against `documentations/api_e+.html` and collected every poll cycle.
The full mapping lives in [API.md → EM6 endpoint coverage](API.md#em6-endpoint-coverage).

| Endpoint | Used for |
|----------|---------|
| `/self/information` | Firmware version, device type, platform HW |
| `/self/ipconfig` | Hostname, IP, MAC, DHCP state |
| `/self/system` | Core temp, fan speed, core voltage, uptime |
| `/self/firmware` | Firmware bank slots (active/default) |
| `/self/license` | Licensed feature map |
| `/self/diag/refclk` | PTP lock status, offset, mean delay, sync counters |
| `/self/diag/ethernet` | Control-plane TX/RX packet counters + RX errors |
| `/self/diag/common` | Video bandwidth usage, watchdog, IPv4 packet drops |
| `/self/interfaces` | Per-interface (e1/e2) IP, gateway, DHCP, VLAN |
| `/lldp` | Discovered LLDP neighbour |
| `/telemetry/node` | Health summary + PTP/refclk status |
| `/telemetry/ports` | Per-port SFP temperature, TX/RX power |
| `/telemetry/devices` | Media-flow packet counters + validity |
| `/port/{id}` | Full SFP DDM with alarm/warning thresholds |
| `/sdi` | SDI operating bit rate |

---

## Decisions

### Reachability uses TCP connect, not ICMP
**Decision:** Independent dual-path reachability (`reachable_red` / `reachable_blue`)
is implemented with a **TCP connect** to the device management port, not a raw
ICMP echo.

**Why:** Raw ICMP echo sockets require elevated privileges — administrator on
Windows, `CAP_NET_RAW` on Linux. The dashboard is designed to run as an
unprivileged process (including the non-root container user). A TCP connect to
the management port exercises the same L3 path and additionally confirms the
port is open, without raw sockets.

**Toggle:** `polling.icmp_enabled` (default `true`) gates the dual-path probe.
The field keeps the `icmp` name for config-compatibility; the implementation is
TCP. If true ICMP is ever required, add it behind the same flag and document the
privilege requirement here.

### Config-plane endpoints are intentionally not polled
**Decision:** The media-routing/configuration endpoints — `/sources`,
`/receivers`, `/senders`, `/flows`, `/sdp`, `/route`, `/clean_switch`,
`/black_burst`, `/sdi_input`, `/sdi_audio`, `/sdi_output`, `/self/protocols`,
`/self/syslog`, `/self/static_route`, `/self/diag/dns` — are **not** part of the
monitoring poll.

**Why:** They describe how media flows are *configured*, not device *health*.
Polling them on every cycle adds load and surfaces configuration state the
monitoring product does not act on. They are reserved for Phase 4 (configuration
management), where they will be read and written on demand, not on a timer.

### Best-effort polling
Every endpoint except `/self/information` is fetched best-effort: a failure
(e.g. the device type does not implement `/sdi`) is tolerated and does not abort
the poll or mark the device unhealthy. Only `/self/information` must succeed for
a device to count as reachable.

---

## Deferred to Later Phases

| Feature | Reason | Planned Phase |
|---------|--------|--------------|
| Device reboot | Destructive — requires confirmation dialog + audit log | Phase 4 |
| Config reset | Destructive | Phase 4 |
| IP reconfiguration | PUT `/self/ipconfig` causes device reboot | Phase 4 |
| Firmware upgrade | Requires multipart upload + slot management | Phase 4 |
| Syslog / DNS / protocols config | Config-plane writes | Phase 4 |
| Media flow routing | `/route`, `/flows`, `/sdp` writes | Phase 4 |

---

## Open Limitations

### Authentication
The emSFP API (`/emsfp/node/v1`) does not require authentication per
`api_e+.html`. The `auth-usvc.json` file documents a separate NEP Broadcast
Control authentication microservice, unrelated to direct device access. If a
deployment restricts device API access, the mechanism must be discovered against
real hardware and implemented in `EmsfpClient`.

### SFP vendor / model strings
The API returns SFP type (`sfp_type`) but not vendor name, part number, or
serial of the transceiver in the documented endpoints. Needs verification
against real hardware (`/self/diag/devices`, `/port/{id}`) to confirm whether
vendor strings are exposed.

### Media-flow telemetry depth
`/telemetry/devices` is parsed into per-device flow counts and validity. The
underlying structure is device-type-specific (encap vs. decap vs. UDC); the
dashboard summarises flow count and total packets rather than modelling every
engine/essence. Deeper per-essence breakdown is a future enhancement.

### Historical data volume
Polling every 30 seconds generates ~2,880 rows/day per device. A daily pruning
job (`polling.history_retention_days`, default 30) bounds growth; SQLite WAL mode
and indexed queries keep queries fast within the window.

### Authentication & RBAC (Phase 5) — disabled by default
Authentication ships **off** (`auth.enabled: false`) so existing/live deployments
keep working with no login. Enabling it requires a `jwt_secret`; on first start an
admin account is seeded (password from `auth.admin_password`, or a random one that
is logged once). RBAC (viewer/operator/admin) is enforced server-side — the
frontend role-gating is convenience only.

### Deferred enterprise items
The following Phase 5 items are intentionally **not** implemented yet, with rationale:
- **LDAP / Active Directory** — requires environment-specific directory config and
  a test directory; local accounts cover the immediate need.
- **PostgreSQL / multi-writer** — SQLite (WAL, single writer) is sufficient at the
  current fleet scale; a Postgres driver swap is isolated to `pkg/database`.
- **Refresh-token rotation** — short-lived bearer tokens (configurable TTL) are
  used; rotation can be layered on without API changes.
- **Scheduled PDF reports** — backlog; CSV export + webhooks cover current reporting.
