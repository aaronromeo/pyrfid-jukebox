# btconnect Resilience Fix — Design

**Date:** 2026-10-16
**Status:** Approved (300s backoff cap)
**Background:** Incident write-up in the handoff at
`/tmp/opencode/pyrfid-jukebox-btconnect-handoff.md` (2026-10-16).

## Problem

`btconnect.Service.Run()` returns on the first `CheckAndConnect` error.
Supervisor's `autorestart=true` then restarts the process, which fails
fast again — a 1–2s connect-storm that wedged the Pi's BCM43430
Bluetooth stack, exhausted `startretries`, and left the service FATAL
so nothing recovered when the speaker returned. A temporary speaker
absence became a permanent outage.

## Goal

The service stays running through connect failures, retries with
bounded exponential backoff, and reconnects on its own when the speaker
returns — proven by unit tests.

## Decisions (confirmed with user)

1. **Bounded exponential backoff** on consecutive failures.
2. **Missing `PJ_BLUETOOTH_DEVICE` stays fatal** (exit). The env var is
   loaded once at process start (`godotenv` in `cmd/main.go`), so a
   retry loop can never pick up a newly set value; FATAL is the correct
   operator signal for a deploy misconfig.
3. **No zombie-connection detection/self-clear** in this change. The
   connect-storm is what re-wedged the stack; backoff prevents storms.
   Revisit only if the failure mode recurs post-fix.

## Design

### Loop behavior (`system/soundsprout/btconnect/service.go`)

`Run(ctx context.Context) error`:

- Each cycle calls `CheckAndConnect()`.
- **On failure:**
  - If `errors.Is(err, ErrMissingDevice)`, return the error (fatal).
  - Otherwise log at Warn —
    `logger.Warn("bluetooth connect failed", "error", err, "failures", n, "retry_in", d)` —
    increment the consecutive-failure count, and sleep the backoff delay.
- **On success:** reset the consecutive-failure count. If there had
  been one or more consecutive failures, log at Info
  (`"bluetooth reconnected"`, with the failure count). Steady-state
  success stays silent (today's behavior).
- **Sleep** is context-aware: default implementation is
  `select { case <-time.After(d): case <-ctx.Done(): return ctx.Err() }`.

### Backoff

`min(15s × 2^(n−1), 300s)` where `n` is the consecutive-failure count:

| failures | delay |
|---|---|
| 1 | 15s |
| 2 | 30s |
| 3 | 60s |
| 4 | 120s |
| 5 | 240s |
| ≥6 | 300s (cap) |

Base interval is the existing `DefaultSleepTime` (15s), reused after
success. Worst-case reconnect latency after the speaker returns: ≤5
minutes. Implemented as an unexported pure helper; covered indirectly
through recorded sleep durations in `Run` tests (no internal test
package).

### API changes

- `NewBtConnectService(runner bluetooth.Runner, logger *slog.Logger, opts ...Option) *Service`
  — backward-compatible variadic; the `main.go` call site is unchanged
  apart from passing `ctx` to `Run`.
- `Option` / `WithSleep(fn func(ctx context.Context, d time.Duration) error)`
  — the only option; the test seam. No `WithBackoff` (YAGNI: tests
  assert the durations passed to the sleep function directly).
- Exported sentinel `ErrMissingDevice`. `CheckAndConnect` returns it
  (wrapped with `%w`) for the missing-env case; its other behavior and
  existing tests are unchanged.
- `cmd/main.go`: `connectService.Run(context.Background())`. No signal
  plumbing — supervisor's SIGTERM kills the process regardless.

### Shared primitives untouched

`bluetooth.Connect` / `Reconnect` still return errors; the `portal`
package depends on that. The fix lives entirely in the `btconnect`
loop.

### Tests (`system/soundsprout/btconnect/service_test.go`)

- Extend `fakeRunner` with `runErrs map[string]error` so
  `bluetoothctl connect <mac>` can fail deterministically.
- All tests pass a discard `slog.Logger`
  (`slog.New(slog.NewTextHandler(io.Discard, nil))`) — the current
  `nil` logger would panic once `Run` logs.
- Fake sleep records delays and cancels the context after a
  configured number of calls, so tests never wait on real timers.
- Cases:
  - `TestRunMissingEnvFatal` — empty env; `Run` returns an error
    wrapping `ErrMissingDevice`; sleep never called.
  - `TestRunBackoffSequence` — connect fails for 6 cycles; recorded
    delays `[15s, 30s, 60s, 120s, 240s, 300s]`.
  - `TestRunResetsBackoffOnSuccess` — fail, fail, success, fail;
    recorded delays `[15s, 30s, 15s, 15s]` (the success cycle sleeps
    the base 15s, and the next failure starts from 15s again).

## Documentation updates

- `README.md` — btconnect description (currently "Every 15s it
  checks…") gains the retry/backoff behavior.
- `docs/rfid-map-and-bluetooth.md` — update the btconnect section;
  add this failure mode (absent speaker → backoff, self-heal) with a
  pointer to the operational runbook in the handoff.
- `AGENTS.md` — update the "loops every 15s" summary line.

## Out of scope

- Zombie-connection detection or radio self-reset (`hcitool dc`,
  `hciconfig reset`) — deferred, would need sudoers for user `pi`.
- Supervisor config changes — `autorestart=true` stays as a safety
  net; `startretries` is no longer reachable via connect failures.
- Signal-based graceful shutdown.
- Any change to the shared `bluetooth` package or `portal`.

## Verification

- `make test` from repo root.
- `golangci-lint run --config system/soundsprout/.golangci.yml ./...`
  from `system/soundsprout/`.
- `pre-commit run --all-files`.
- On-hardware validation (coordinated with the user, after deploy):
  power off the Esinkin → `btconnect` stays RUNNING (no FATAL, no
  storm), `hcitool con` stays clean → power it on → auto-reconnect
  within one backoff cycle. Pi: `ssh pi@192.168.68.89`.

## Deployment

Push to `main`; `gitpull.sh` rebuilds the binary, installs it to
`/usr/local/bin/`, and restarts `btconnect` and `portal`.
