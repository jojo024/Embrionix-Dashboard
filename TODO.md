# TODO

Active work items. See [ROADMAP.md](ROADMAP.md) for the full phased plan.

---

## Phase 2 — Next Up

- [ ] #1 — Dual-path reachability tracking in polling engine (Red + Blue independently)
- [ ] #2 — ICMP ping check (separate from API poll) — evaluate `golang.org/x/net/icmp`
- [ ] #3 — Fleet-wide alarm panel on Dashboard landing page
- [ ] #4 — Alarm deduplication in `PollResult` storage
- [ ] #5 — `self/diag/ethernet` stats — TX/RX packet counters per interface
- [ ] #6 — LLDP neighbour display on Interfaces tab
- [ ] #7 — SDI signal presence (for encap/decap devices) on Interfaces tab
- [ ] #8 — Auto-refresh countdown indicator in UI
- [ ] #9 — History pruning background job with configurable retention

## Infrastructure

- [ ] Move Google Fonts `@import` to `index.html` `<link>` to fix PostCSS warning
- [ ] Add code splitting (dynamic imports) for recharts to reduce initial bundle size
- [ ] Add `go test` unit tests for `DeviceService` and `PollingService`
- [ ] Add frontend Vitest unit tests for API client and hooks

## Phase 1 Polish

- [ ] Toast notifications for create/update/delete success and errors
- [ ] Keyboard shortcut to open "Add Device" (e.g. `N`)
- [ ] Empty-state illustration on Dashboard when 0 devices
- [ ] Favicon and `<title>` update

---

## Completed

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
