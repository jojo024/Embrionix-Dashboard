# Network & Device Impact Analysis

This document quantifies the load the dashboard places on (a) each **EM6 device's
embedded HTTP server** and (b) the **management network**, and the design choices
that keep that impact minimal. The guiding principle: *the device is the
constrained party* — its small embedded web server, not the dashboard host, is
what we protect.

> TL;DR — In steady state the dashboard makes **~6 lightweight HTTP GETs per
> device every 30 s** (down from ~17), reuses a **single TCP connection** per
> poll instead of one-per-request, and **bounds fleet-wide concurrency** so a
> large fleet never bursts. All device writes are user-initiated only.

---

## 1. What touches a device

The dashboard interacts with a device in exactly four ways. Three are read-only;
the fourth only happens on an explicit operator action.

| Interaction | Trigger | Frequency | Cost to device |
|-------------|---------|-----------|----------------|
| Health poll (tiered) | background | every `interval_seconds` (30 s) | ~6 GETs (light) / ~17 (full) |
| Dual-path reachability probe | background | every cycle | 1 TCP connect per Red/Blue IP |
| On-demand poll ("Poll Now") | user click | rare | one full poll (~17 GETs) |
| Config write / reboot / reset | user click + confirm | rare | one PUT |

There is **no automatic writing to devices, ever** — configuration changes,
reboots and resets are only sent in response to a confirmed operator action and
are recorded in the audit log.

---

## 2. The polling load, quantified

A *full* poll reads every relevant endpoint (~17 GETs for a 2-port device):

```
fast (always):  /self/information  /self/system  /telemetry/node
                /telemetry/ports   /self/diag/refclk  /self/diag/common      = 6
slow (full):    /self/ipconfig  /self/firmware  /self/license
                /self/diag/ethernet  /self/interfaces  /lldp
                /telemetry/devices  /sdi  /port  /port/{id}×N                 = ~11
```

Most of that data is **static or slow-moving** (firmware banks, licenses,
interface config, LLDP neighbour, SDI bitrate, SFP module identity). Re-fetching
it every 30 s is wasted load on the device.

### Tiered polling

The poller therefore splits work into two tiers:

- **Fast tier — every cycle (~6 GETs):** the dynamic health signals an operator
  watches in real time (temperature, fan, SFP TX/RX power, PTP lock & offset,
  video-bandwidth health).
- **Slow tier — every `full_every` cycle (default 10):** the static/heavy
  endpoints, including the per-port SFP DDM detail (the single heaviest part, one
  request per port). Between full polls these fields are **carried forward** from
  the last full poll, so the UI still shows them; alarms are re-derived from the
  merged data each cycle so they stay correct.

### Steady-state request rate (per device)

| | Requests / 30 s cycle | Requests / minute |
|---|---|---|
| **Before tiering** | 17 every cycle | **~34** |
| **After tiering** (`full_every: 10`) | 6 light, 17 on every 10th | **~14** |

That's a **~58 % reduction** in requests hitting each device's web server, with
the full dataset still refreshed every ~5 minutes (10 × 30 s).

---

## 3. TCP connection behaviour

Previously the HTTP client ran with keep-alive **disabled**, so every one of the
~17 GETs opened and tore down its own TCP connection — ~34 connection setups per
device per minute. Embedded device web servers typically allow only a handful of
concurrent connections, so this churn is the most likely way to stress one.

The client now uses **HTTP keep-alive within a poll** (`MaxConnsPerHost: 2`,
`IdleConnTimeout: 5 s`): the many GETs of a single poll reuse one connection,
which then closes shortly after so it doesn't hold a device socket between
cycles.

| | TCP handshakes / device / minute |
|---|---|
| Before (keep-alive off) | ~34 |
| After (keep-alive within poll) | ~4 |

≈ **88 % fewer TCP handshakes** on the device.

---

## 4. Fleet-wide burst control

`pollAll()` previously fanned out **all** devices concurrently, so a 100-device
fleet fired 100 simultaneous poll bursts at the top of every cycle — a spike on
the switch fabric and on the dashboard host.

Polls are now run through a **bounded worker pool** (`max_concurrent_polls`,
default 8). At most N device polls run at once; the rest queue. This smooths the
per-cycle network profile from a single tall spike into a steady, low plateau,
at the cost of taking slightly longer to work through a very large fleet (which
is fine — health data 30 s old is still timely).

---

## 5. Bandwidth

Responses are small JSON documents (≈0.5–4 KB each). A full poll transfers on the
order of **30–40 KB**; a light poll **~10 KB**. Per device per minute that is
roughly **25–30 KB** — negligible on a management LAN. The dashboard never
streams media or touches the data plane.

---

## 6. Dashboard ↔ backend load (does *not* touch devices)

The browser polls the **dashboard backend** (not the devices) on timers:
device list / summary / alarms every 30 s, history every 60 s, and a small
24-point sparkline per device card every 60 s. These are served from the local
SQLite cache (indexed reads) and the in-memory poll-state cache — the live device
status is served from memory with **no device round-trip**. The per-card
sparkline is an N-queries-per-fleet pattern against SQLite; it is cheap today and
a batch endpoint is noted as a future optimisation for very large fleets.

History storage is bounded by the daily pruning job
(`history_retention_days`, default 30); at a 30 s interval that's ~2,880 rows/day
per device, kept fast by WAL mode and indexed queries.

---

## 7. Tuning for minimal impact

All knobs live under `polling:` in `config.yaml` (or `EMB_POLLING_*` env vars):

| Setting | Default | Effect on device load |
|---------|---------|-----------------------|
| `interval_seconds` | 30 | Linear: 60 s halves all polling load |
| `full_every` | 10 | Higher = the heavy slow tier runs less often |
| `max_concurrent_polls` | 8 | Lower = gentler, smoother fleet-wide bursts |
| `icmp_enabled` | true | Set false to drop the 2 extra TCP probes/cycle |
| `monitoring_enabled` (per device) | true | Excludes a device from polling entirely |

**Recommended profiles:**

- **Default / small fleet (≤ ~30 devices):** ship defaults are already light.
- **Large fleet (100s of devices):** `interval_seconds: 60`, `full_every: 20`,
  `max_concurrent_polls: 5` — roughly a quarter of the default per-device load,
  spread thinly across the network.
- **Sensitive / production-critical devices:** raise `interval_seconds` and/or set
  `monitoring_enabled: false` on devices that must not be touched on a schedule;
  use **Poll Now** for on-demand checks instead.

---

## 8. Summary of guarantees

- Steady-state: **~6 small GETs over ~1 reused TCP connection per device per
  30 s**; full refresh every ~5 min.
- **No background writes** to devices — only confirmed, audited operator actions.
- Fleet bursts are **bounded** and tunable; every load parameter is configurable
  and can be dialled down further without code changes.
- The dashboard host absorbs the heavier work (caching, history, fan-out); the
  device sees only a light, polite read pattern.
