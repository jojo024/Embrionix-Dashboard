# TODO

Active work items (features and bug fixes).

---

## Phase 2 — Remaining

- [ ] #2 — True ICMP echo behind `polling.icmp_enabled` (currently TCP-connect; needs `CAP_NET_RAW`/admin) — see [ISSUES.md](ISSUES.md)
- [ ] #4 — Alarm deduplication / de-flap in `PollResult` storage
- [ ] SFP vendor / part / serial strings (pending real-hardware verification)
- [ ] Deeper per-essence media-flow breakdown from `/telemetry/devices`

## Phase 3 — Remaining

- [ ] Email (SMTP) alerting in addition to webhooks
- [ ] Per-device threshold overrides (currently fleet-wide via config)
- [ ] Recharts zoom/pan on monitoring charts
- [ ] SFP optical-power degradation detection
- [ ] Fleet temperature heatmap

## Infrastructure

- [ ] Add frontend component tests (jsdom) for the alarm panel and toast system
- [ ] Add `go test` coverage for handlers (httptest) and the emSFP client (mock server)
- [ ] Make alerting thresholds editable at runtime (persist to app_settings + reload poller)

## Phase 1 Polish

- [ ] Empty-state illustration on Dashboard when 0 devices (currently text only)
- [ ] Surface the **N** shortcut hint in the Add Device button tooltip

---

## Completed

### Phase 3 (v0.3.0)
- [x] Configurable alert thresholds (temperature, response time)
- [x] Status-transition detection + alert history (`alert_events`)
- [x] Webhook notifications (Slack-compatible / generic)
- [x] `GET /api/v1/alerts` history + `GET /api/v1/config` effective config
- [x] Device-card temperature sparklines (pure SVG)
- [x] CSV export of poll history
- [x] Settings → Alerting tab; status history in device Logs tab
- [x] Alert history pruned with the retention job
- [x] Notifier + threshold unit tests

### Phase 2 (v0.2.0)
- [x] Full EM6 endpoint coverage (PTP, firmware, license, ethernet, common, interfaces, LLDP, telemetry/devices, SDI)
- [x] #1 — Dual-path reachability tracking (Red + Blue independent, TCP probe)
- [x] #3 — Fleet-wide alarm panel on the Dashboard
- [x] #5 — `self/diag/ethernet` TX/RX packet counters
- [x] #6 — LLDP neighbour display on Interfaces tab
- [x] #7 — SDI signal/bit-rate on device detail
- [x] #8 — Auto-refresh countdown indicator
- [x] #9 — History pruning background job with configurable retention
- [x] PTP offset trend chart
- [x] Toast notifications for CRUD + on-demand poll
- [x] Keyboard shortcut (N) to add a device
- [x] API status indicator verifies real connectivity
- [x] Move Google Fonts `@import` to `index.html` (PostCSS warning fixed)
- [x] Code-split recharts + lazy-load heavy routes
- [x] Favicon and `<title>` update
- [x] Go unit tests (status, PTP decode, fleet alarms) + Vitest frontend test

### Phase 1 (v0.1.0)
- [x] Project scaffold (Go + React)
- [x] Device CRUD API and UI
- [x] Background polling engine
- [x] emSFP API client (information, system, telemetry, SFP DDM)
- [x] Dashboard — card + table views
- [x] Device detail — 5 tabs
- [x] Monitoring page
- [x] Settings page
- [x] GitHub Actions CI + release
- [x] Dockerfile + docker-compose
- [x] Documentation suite
