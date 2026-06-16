# Known Issues and API Limitations

This file documents features that could not be implemented as desired due to emSFP API constraints, and proposes alternatives where possible.

---

## Confirmed Supported

The following emSFP API endpoints have been verified against `documentations/api_e+.html` and are used in Phase 1:

| Endpoint | Method | Used for |
|----------|--------|---------|
| `/self/information` | GET | Firmware version, device type |
| `/self/ipconfig` | GET | Hostname, IP, MAC, DHCP state |
| `/self/system` | GET | Core temp, fan speed, core voltage, uptime |
| `/telemetry/node` | GET | Health summary + PTP/refclk status |
| `/telemetry/ports` | GET | Per-port SFP temperature, TX/RX power |
| `/port/{id}` | GET | Full SFP DDM with alarm/warning thresholds |

---

## Deferred to Later Phases

| Feature | Reason | Planned Phase |
|---------|--------|--------------|
| Device reboot | Destructive — requires confirmation dialog + audit log | Phase 4 |
| Config reset | Destructive | Phase 4 |
| IP reconfiguration | PUT `/self/ipconfig` causes device reboot | Phase 4 |
| Firmware upgrade | Requires multipart upload + slot management | Phase 4 |
| Syslog configuration | Low priority; no demand yet | Phase 4 |
| VLAN configuration | PUT `/self/ipconfig` — covered with IP config | Phase 4 |

---

## Limitations and Workarounds

### Authentication
**Issue:** The emSFP API (`/emsfp/node/v1`) does not appear to require authentication based on the available documentation (`api_e+.html`). The `auth-usvc.json` file documents a separate NEP Broadcast Control authentication microservice which is unrelated to direct device access.

**Impact:** No token management needed for device polling. If your environment restricts device API access, this will require investigation.

**Action:** If authentication is discovered to be required in testing, document the mechanism and implement it in the `EmsfpClient`.

### ICMP Reachability
**Issue:** Raw ICMP (ping) requires elevated OS privileges on Linux and is non-trivial on Windows without admin rights.

**Workaround:** Phase 1 uses a lightweight HTTP GET to `/self/information` as the reachability check. This is more meaningful than ICMP because it confirms the device API is responding, not just that it's pingable.

**Phase 2:** Evaluate `golang.org/x/net/icmp` with capability setting (`CAP_NET_RAW`) on Linux and investigate Windows alternatives.

### Dual-Path Status (Red / Blue)
**Issue:** The polling engine currently polls only the Red IP (falling back to Blue if Red is unset). Independently tracking both paths requires two concurrent polls per device.

**Phase 2:** The `reachability` endpoint already supports dual-path checking on demand. Extend `PollingService` to store `reachable_red` and `reachable_blue` independently per poll cycle.

### Flow Telemetry
**Issue:** `/telemetry/devices` returns per-device/engine/flow packet counts. The data structure is device-type-specific (encapsulator vs. decapsulator) and complex.

**Phase 2:** Parse flow telemetry and surface packet count trends in the Monitoring tab.

### SFP Vendor / Model Information
**Issue:** The emSFP API returns SFP type (`sfp_type`, `detected_sfp_type`) but not the vendor name, part number, or serial number of the transceiver module directly in the documented endpoints.

**Action:** Investigate `/self/diag/devices` and `/port/{id}` more thoroughly against real hardware to find if vendor strings are available.

### SDI Input / Output Status
**Issue:** `/sdi_input/{id}` and `/sdi_output/{id}` endpoints for SDI signal presence and format detection are present in the API but not yet parsed.

**Phase 2:** Add SDI signal presence to the Interfaces tab (relevant for encapsulator/decapsulator devices).

### Historical Data Volume
**Issue:** Polling every 30 seconds generates ~2,880 rows/day per device. With 20 devices over 90 days, that is ~5.2 million rows.

**Mitigation:** SQLite WAL mode and indexed queries keep this manageable. Phase 3 will add a background pruning job with configurable retention (default 30 days).
