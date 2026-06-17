# GitHub Issues Implementation Plan

**Last Updated:** 2026-06-17  
**Total Issues:** 10 (1 Bug blocking others, 9 Features)

---

## Priority Breakdown

### 🔴 BLOCKING BUG (Must Fix First)
- **#28** - Remote access: "Last polled time" shows negative numbers when accessing from non-localhost
  - **Impact:** High - breaks remote access usability
  - **Estimated effort:** 2-4 hours (likely timezone/time calculation issue)
  - **Blocker for:** User experience improvements depend on this working correctly

---

### 🟡 High Priority (Phase 1 - UX/Display)
These improve the immediate user experience without complex backend changes.

1. **#31** - Uptime in devices overview (Bug)
   - **Summary:** Uptime metric is not displayed on device cards
   - **Impact:** Missing telemetry data user expects
   - **Effort:** 1-2 hours (backend: already polled; frontend: display only)
   - **Files:** `web/src/components/DeviceCard.tsx`, `internal/models/device.go`

2. **#33** - Firmware translation (Feature)
   - **Summary:** Convert hex firmware values (0x5fce17b0) to readable format (3.4.1607342000)
   - **Impact:** Improves readability on device cards
   - **Effort:** 2-3 hours (backend: parsing logic; frontend: display)
   - **Notes:** Need to understand firmware format; may need lookup table or algorithm
   - **Verification:** Against real device firmware values

3. **#32** - Devices fixed header (Feature)
   - **Summary:** Sticky header when viewing device details/interfaces for easier refresh access
   - **Impact:** UX improvement when scrolling through device details
   - **Effort:** 2-3 hours (CSS + React component state)
   - **Files:** `web/src/pages/DeviceDetail.tsx`

---

### 🟠 Medium Priority (Phase 2 - Data Quality & Org)
These add validation and quality-of-life features.

4. **#34** - Flag rx_power 0 (Feature)
   - **Summary:** Warning flag when SFP power values are 0 on active interfaces
   - **Impact:** Early warning for configuration/hardware issues
   - **Effort:** 3-4 hours (backend: threshold logic; frontend: warning badge)
   - **Implementation:**
     - Add flag to `DevicePollingData` (e.g., `PowerWarnings []string`)
     - Check in `deriveStatus()` if port is up but power is 0
     - Display as warning badge on device card / interface tab

5. **#21** - Flag mismatched ipconfig (Feature)
   - **Summary:** Compare configured IPs against what device reports; flag mismatches
   - **Impact:** Prevents misconfiguration
   - **Effort:** 3-4 hours (backend: comparison logic; frontend: mismatch indicator)
   - **Implementation:**
     - Backend: Compare `device.ManagementIPRed/Blue` vs `pollingData.IP`
     - Frontend: Badge/indicator when mismatch detected
     - Document in `ISSUES.md` how this handles dual-IP fallback

6. **#30** - Reorder device cards (Feature)
   - **Summary:** Drag-to-reorder device cards in dashboard
   - **Impact:** Improves organization for user's preferred layout
   - **Effort:** 4-6 hours (frontend: drag-drop library + backend: persistence)
   - **Implementation:**
     - Add `DisplayOrder` field to Device model
     - Implement drag-drop (react-beautiful-dnd or built-in)
     - PATCH endpoint to update order
     - Persist to DB; fetch ordered list on GET /devices

---

### 🟢 Lower Priority (Phase 3 - Feature Enhancement)
These add new information or UX improvements but don't fix bugs.

7. **#29** - LLDP info (Feature)
   - **Summary:** Display LLDP neighbor info next to ports 3 & 5 on device card
   - **Impact:** Useful for debugging network topology
   - **Effort:** 2-3 hours (already polled; display only)
   - **Files:** Render `pollingData.LLDP` in InterfacesTab
   - **Notes:** Check API coverage in api_e+.html; may need to parse LLDP response

8. **#27** - Sliding tab (Feature)
   - **Summary:** Collapsible/sliding sidebar to maximize dashboard space
   - **Impact:** UX improvement for smaller screens or focused monitoring
   - **Effort:** 3-4 hours (CSS transitions + React state)
   - **Implementation:**
     - Add sidebar collapse toggle in Layout
     - CSS animations for slide in/out
     - Persist preference in localStorage or user settings

---

### 🔵 Backend Enhancement (Phase 4 - CRUD)
Device management features that require schema/API changes.

9. **#2** - Device add (Feature)
   - **Summary:** Make ManagementIPRed/Blue required; auto-populate SN & firmware from device on creation
   - **Impact:** Prevents incomplete device setup
   - **Effort:** 3-4 hours (validation + API call on save)
   - **Implementation:**
     1. Make IP fields required in frontend form validation
     2. On device create, probe the device via EmsfpClient to fetch:
        - SN (from `/self/information`)
        - Firmware version (from `/self/information` or `/self/firmware`)
     3. Pre-fill form before submission
     4. Show loading state during probe
     5. Handle probe failures gracefully (timeout, unreachable)

---

## Implementation Sequence (Recommended)

```
Week 1:
  [Mon]  Fix #28 (remote access bug)
  [Tue]  Implement #31 (uptime display)
  [Wed]  Implement #33 (firmware translation)
  [Thu]  Implement #32 (fixed header)
  [Fri]  Test & merge #31, #33, #32

Week 2:
  [Mon]  Implement #34 (flag rx_power 0)
  [Tue]  Implement #21 (flag ipconfig mismatch)
  [Wed]  Implement #30 (reorder cards)
  [Thu]  Implement #29 (LLDP display)
  [Fri]  Test & merge

Week 3:
  [Mon-Wed] Implement #27 (sliding sidebar)
  [Thu-Fri] Implement #2 (device add with auto-probe)
  
Final: Code review, integration test, release
```

---

## Development Notes

### Code Quality Checklist
- [ ] `go vet ./...` passes
- [ ] `go build ./...` passes (CGO_ENABLED=0)
- [ ] `npx tsc --noEmit` passes
- [ ] `npm run build` produces clean dist
- [ ] New emSFP endpoints verified against `documentations/api_e+.html`
- [ ] API.md updated with any new endpoints
- [ ] CHANGELOG.md updated with user-visible changes
- [ ] Unit/integration tests for business logic
- [ ] Manual testing on real device (if touching emSFP API)

### Files Likely to Change
**Backend:**
- `internal/models/` — Add fields to DevicePollingData, Device
- `internal/services/polling.go` — Status derivation, flag logic
- `internal/services/emsfp_client.go` — If new endpoints needed
- `internal/api/handlers/` — New endpoints for reordering, etc.
- `internal/repositories/` — Schema updates

**Frontend:**
- `web/src/components/DeviceCard.tsx` — Display improvements (#31, #33, #34, #29)
- `web/src/pages/DeviceDetail.tsx` — Fixed header (#32), tabs
- `web/src/pages/Dashboard.tsx` — Drag-drop reordering (#30), sidebar collapse (#27)
- `web/src/pages/DevicesPage.tsx` — Device form (#2)
- `web/src/types/index.ts` — TypeScript types for new fields

### Known Constraints
- **No CGO** — Keep all Go dependencies pure-Go
- **Minimal device impact** — Existing tiered polling must not change
- **API verification required** — Every new emSFP endpoint must be confirmed in api_e+.html
- **Remote access fix (#28)** — Likely a timezone or time calculation bug in the frontend; check React Query's time helpers and API response timestamps

---

## Testing Strategy

### Per-Issue Verification
1. **#28** — Access dashboard from non-localhost; verify "Last polled" shows positive relative time
2. **#31** — Check device card displays uptime like temp/fan
3. **#33** — Verify hex firmware translates to human-readable version
4. **#32** — Scroll device detail tab; confirm header stays visible
5. **#34** — Set port to up but power to 0; confirm warning appears
6. **#21** — Configure mismatched IP; poll device; verify mismatch flag
7. **#30** — Drag device cards; refresh page; verify order persists
8. **#29** — Check LLDP info displays on interfaces tab
9. **#27** — Collapse sidebar; verify layout adjusts, state persists
10. **#2** — Add device with Red/Blue IPs; verify SN & firmware auto-populated

### Integration Test
- Polling cycle completes without errors
- All new flags/fields serialize correctly to DB
- API response matches TypeScript types
- Frontend renders all states (loading, success, error)

---

## Risk Assessment

| Issue | Risk | Mitigation |
|-------|------|-----------|
| #28 | High — blocks remote users | Test before/after with non-localhost access |
| #31 | Low — display only | Verify backend already sends uptime in polling |
| #33 | Medium — firmware format unknown | Document format in code comment; test with real devices |
| #32 | Low — CSS only | Test on narrow viewport |
| #34 | Medium — new status logic | Verify doesn't interfere with deriveStatus() |
| #21 | Medium — config validation | Handle edge cases (one IP empty, both empty, both same) |
| #30 | Medium — persistence | DB migration? Or new column? Plan before implementing |
| #29 | Low — LLDP already polled | Verify data structure in emsfp response |
| #27 | Low — sidebar collapse | Test on mobile / narrow screens |
| #2 | Medium — async device probe | Timeout? Unreachable? Partial data? |

---

## Documentation Updates Required

- [ ] API.md — New endpoints (reorder, etc.)
- [ ] CHANGELOG.md — All user-visible changes
- [ ] ISSUES.md — Firmware format, ipconfig edge cases
- [ ] CLAUDE.md — If new polling logic added
- [ ] README.md — If new user features warrant mention

---

## Next Steps

1. **Fix #28 immediately** — Remote access is broken; all other improvements depend on user confidence
2. **Batch #31, #33, #32, #29** — All display-only; low risk; high visibility
3. **Roll out #34, #21** — Quality-of-life improvements; moderate complexity
4. **Ship #30** — Popular request; finish with persistence testing
5. **Polish #27, #2** — Convenience features; save for final phase

**Estimated Total Duration:** 3–4 weeks (depending on #28 root cause and testing)
