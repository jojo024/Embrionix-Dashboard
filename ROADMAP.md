# Roadmap

## Phase 1 — Foundation ✅

**Goal:** Working skeleton that lets operators manage a device inventory and see live health at a glance.

- [x] Project structure (Go clean architecture + React/Vite)
- [x] SQLite database with GORM (device inventory, poll history, settings)
- [x] Device CRUD — add, edit, delete, search, disable monitoring
- [x] Background polling engine — concurrent per-device, configurable interval
- [x] emSFP API client — `self/information`, `self/system`, `telemetry/node`, `telemetry/ports`, `port/{id}` DDM
- [x] REST API (Gin) with CORS, request logging, health endpoint
- [x] Dashboard — card + table views, status colour-coding
- [x] Device detail — Overview, Interfaces, SFP, Monitoring charts, Logs tabs
- [x] Monitoring page — fleet-wide table sorted by severity
- [x] Settings — polling config, device management, about
- [x] GitHub Actions — build/test CI, multi-platform release workflow
- [x] Docker support — Dockerfile + docker-compose.yml
- [x] Documentation — README, ARCHITECTURE, API, CONTRIBUTING, etc.

---

## Phase 2 — Monitoring ✅

**Goal:** Rich, reliable monitoring with actionable alarm visibility.

- [x] Reachability probe per device, separate from the API poll (TCP-connect; true ICMP tracked in [TODO.md](TODO.md))
- [x] Dual-path reachability (Red + Blue independently tracked)
- [x] Fleet-wide alarm panel on Dashboard
- [x] `self/diag/ethernet` stats — TX/RX packet counters, error rates
- [x] LLDP neighbour info (`/lldp`) surfaced on Interfaces tab
- [x] PTP/refclk status prominently displayed per device (detailed `/self/diag/refclk`)
- [x] Auto-refresh indicator (countdown to next poll)
- [x] Firmware banks, license features, per-interface config, media-flow telemetry, SDI bit rate
- [ ] Alarm deduplication — don't store the same alarm twice
- [ ] Alarm history table in device Logs tab
- [ ] Device uptime tracking and alerting on unexpected reboots

---

## Phase 3 — Advanced Monitoring ✅

**Goal:** Historical trends and configurable alerting.

- [x] Extended retention settings (configurable days to keep poll history)
- [x] History pruning background job
- [x] Dashboard trend sparklines per device card
- [x] Export history data as CSV
- [x] Webhook notifications (Slack-compatible / generic) on status transitions
- [x] Configurable alert thresholds (temperature, response time)
- [x] Status-transition history (alert log) per device
- [ ] Email alerting (SMTP) when device transitions to critical/offline
- [ ] Per-device threshold overrides (currently fleet-wide)
- [ ] Recharts zoom/pan on monitoring charts
- [ ] SFP optical power degradation detection
- [ ] Fleet temperature heatmap

---

## Phase 4 — Configuration Management

**Goal:** Read and write device configuration safely through the dashboard.

### Phase 4a — Read-only views ✅
- [x] View full device IP configuration (`/self/ipconfig`)
- [x] View system config (`/self/system` — staging, min fan, ST 2022-7 class)
- [x] View protocols (`/self/protocols` — mDNS, Ember+, SAP)
- [x] View syslog configuration and monitoring events (`/self/syslog`)
- [x] View static routes (`/self/static_route`)
- [x] View DNS (`/self/diag/dns`)
- [x] Configuration tab on device detail (on-demand fetch, read-only)

### Phase 4b — Writes (for initial device setup) ✅
- [x] Change management IP (static / DHCP toggle) with confirmation dialog
- [x] VLAN configuration (`ctl_vlan_id`, `ctl_vlan_pcp`, `ctl_vlan_enable`)
- [x] Protocols write (mDNS, Ember+, SAP)
- [x] Syslog server configuration write (`/self/syslog`)
- [x] Static routes write (`/self/static_route`)
- [x] Device reboot action with confirmation dialog
- [x] Config reset actions (flows / application / generic / system)
- [x] Audit log of all configuration writes and actions
- [x] Configuration backup — export device config as a JSON snapshot
- [x] Configuration restore — push a saved snapshot back to the device
- [x] Database backup — consistent SQLite snapshot (`VACUUM INTO`) download
- [x] Bulk configuration — apply syslog/protocols to multiple devices at once
- [ ] Database in-place restore (intentionally manual: stop server, swap file)

---

## Phase 5 — Enterprise Features ✅ (core)

**Goal:** Multi-user, auditable, and notification-ready.

- [x] Local user accounts with hashed passwords (bcrypt)
- [x] JWT authentication for the REST API
- [x] Role-based access control — Viewer / Operator / Admin roles
- [x] Audit log — configuration changes recorded (user attribution when auth on)
- [x] API key support for external integrations (`X-API-Key`)
- [x] Read-only access mode (Viewer role / auth-disabled implicit admin)
- [ ] LDAP / Active Directory authentication — *deferred (see ISSUES.md)*
- [ ] Multi-user concurrency / PostgreSQL backend — *deferred (SQLite single-writer is sufficient at current scale)*
- [ ] Refresh-token rotation — *deferred (short-lived bearer tokens for now)*
- [ ] Scheduled reports (daily/weekly PDF summary) — *deferred (backlog)*

---

## Backlog / Ideas

- [x] Ansible inventory export (dynamic-inventory JSON, group `emsfp`)
- [x] Keyboard shortcuts for power users (`g`+key navigation, `?` help)
- [x] Minimal-impact polling (tiered fetch, keep-alive, bounded concurrency) — see [PERFORMANCE.md](PERFORMANCE.md)
- [ ] Mobile-optimised layout — responsive polish (partial; sidebar already collapses)
- [ ] Dark/light theme toggle — *deferred: NOC/SOC dashboards are intentionally dark; a full light theme needs a colour-token refactor for low value*
- [ ] NMS integration — SNMP trap receiver / forwarder — *deferred: adds a listener and ongoing network/device load, which runs counter to the minimal-impact goal; revisit only if required*
- [ ] Automated SFP vendor/model lookup from serial — *deferred: the documented API exposes `sfp_type` but not vendor/PN/serial strings (see ISSUES.md); needs hardware verification first*
- [ ] Embrionix firmware upgrade workflow — *deferred: high-risk write path to live hardware; needs a staged rollout design + non-production validation before implementing*
