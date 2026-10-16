# btconnect Resilience Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `btconnect.Service.Run` survive connect failures — log them, retry with bounded exponential backoff (15s → 300s cap, reset on success), and keep running so the jukebox self-heals when the speaker returns.

**Architecture:** The fix lives entirely in the `btconnect` package. `Run` gains a `context.Context` parameter, a consecutive-failure counter, and an injectable sleep function (the test seam). Missing `PJ_BLUETOOTH_DEVICE` stays fatal via an exported sentinel error. The shared `bluetooth` package and `portal` are untouched.

**Tech Stack:** Go 1.21, `log/slog`, testify (`assert`/`require`). Module `aaronromeo.com/soundsprout`. Spec: `docs/superpowers/specs/2026-10-16-btconnect-resilience-design.md`.

## Global Constraints

- Do not modify `system/soundsprout/bluetooth/` (shared with `portal`, which needs error returns) or `system/soundsprout/portal/`.
- No new module dependencies.
- Go tests live in external package `btconnect_test` (`package btconnect_test` at top of `service_test.go`) and use testify.
- Test suite must pass a real `*slog.Logger` (a `nil` logger panics once `Run` logs); use a discard handler in tests.
- Backoff schedule: 15s, 30s, 60s, 120s, 240s, then capped at 300s; base interval after success is 15s (`DefaultSleepTime`).
- Verification commands (run from repo root): `make test`, `golangci-lint run --config system/soundsprout/.golangci.yml ./...` (from `system/soundsprout/`), `pre-commit run --all-files`.
- Commit style: conventional commits, lowercase, scope in parentheses (matches existing history, e.g. `fix(gitpull): ...`).

---

### Task 1: ErrMissingDevice sentinel error

**Files:**
- Modify: `system/soundsprout/btconnect/service.go` (add sentinel, return it from `CheckAndConnect`)
- Test: `system/soundsprout/btconnect/service_test.go` (`TestCheckAndConnectMissingEnv`)

**Interfaces:**
- Produces: `btconnect.ErrMissingDevice` (`error`, exported sentinel) — Task 2's `Run` matches on it with `errors.Is`.

- [ ] **Step 1: Write the failing test**

In `system/soundsprout/btconnect/service_test.go`, replace `TestCheckAndConnectMissingEnv` with:

```go
func TestCheckAndConnectMissingEnv(t *testing.T) {
	t.Setenv("PJ_BLUETOOTH_DEVICE", "")
	svc := btconnect.NewBtConnectService(&fakeRunner{}, nil)
	err := svc.CheckAndConnect()
	require.Error(t, err)
	assert.ErrorIs(t, err, btconnect.ErrMissingDevice)
}
```

Add `"errors"` is not needed in the test file (`assert.ErrorIs` suffices). Imports stay: `testing`, `aaronromeo.com/soundsprout/btconnect`, `assert`, `require`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd system/soundsprout && go test ./btconnect/ -run TestCheckAndConnectMissingEnv -v`
Expected: FAIL — `undefined: btconnect.ErrMissingDevice`

- [ ] **Step 3: Write minimal implementation**

In `system/soundsprout/btconnect/service.go`:

Replace the import block (`"fmt"` is no longer used):

```go
import (
	"errors"
	"log/slog"
	"os"
	"time"

	"aaronromeo.com/soundsprout/bluetooth"
)
```

After `const DefaultSleepTime = 15`, add:

```go
// ErrMissingDevice reports that PJ_BLUETOOTH_DEVICE is not set. It is
// fatal: the environment is loaded once at process start, so retrying
// cannot help.
var ErrMissingDevice = errors.New("env var PJ_BLUETOOTH_DEVICE not set")
```

In `CheckAndConnect`, replace the missing-env branch:

```go
	if device == "" {
		return ErrMissingDevice
	}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd system/soundsprout && go test ./btconnect/ -v`
Expected: PASS (both existing tests; `TestCheckAndConnectConnects` is unchanged)

- [ ] **Step 5: Commit**

```bash
git add system/soundsprout/btconnect/service.go system/soundsprout/btconnect/service_test.go
git commit -m "fix(btconnect): make missing device env a sentinel error"
```

---

### Task 2: Run(ctx) survives connect failures (injectable sleep)

**Files:**
- Modify: `system/soundsprout/btconnect/service.go` (sleep seam, `Run(ctx)`)
- Modify: `system/soundsprout/cmd/main.go:36-49` (pass `context.Background()`)
- Test: `system/soundsprout/btconnect/service_test.go` (fake sleep, discard logger, `runErrs` on `fakeRunner`)

**Interfaces:**
- Consumes: `ErrMissingDevice` (Task 1).
- Produces:
  - `btconnect.SleepFunc` = `func(ctx context.Context, d time.Duration) error`
  - `btconnect.WithSleep(fn SleepFunc) Option` — functional option on `NewBtConnectService(runner, logger, opts ...Option)`
  - `Run(ctx context.Context) error` — returns `ErrMissingDevice` (fatal), the underlying sleep/ctx error, or loops forever.
  - Test helpers `recordingSleep` and `newDiscardLogger` in `service_test.go`, reused by Task 3.

- [ ] **Step 1: Write the failing tests**

In `system/soundsprout/btconnect/service_test.go`, replace the entire file with:

```go
package btconnect_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"aaronromeo.com/soundsprout/btconnect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRunner struct {
	outputs map[string]string
	runErrs map[string]error
	runs    []string
}

func (f *fakeRunner) Output(name string, args ...string) (string, error) {
	key := name
	for _, a := range args {
		key += " " + a
	}
	return f.outputs[key], nil
}

func (f *fakeRunner) Run(name string, args ...string) error {
	key := name
	for _, a := range args {
		key += " " + a
	}
	f.runs = append(f.runs, key)
	return f.runErrs[key]
}

// recordingSleep records delays and cancels the loop's context once
// stop sleeps have been recorded.
type recordingSleep struct {
	delays []time.Duration
	stop   int
	cancel context.CancelFunc
}

func (r *recordingSleep) sleep(ctx context.Context, d time.Duration) error {
	r.delays = append(r.delays, d)
	if len(r.delays) >= r.stop {
		r.cancel()
	}
	return ctx.Err()
}

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCheckAndConnectMissingEnv(t *testing.T) {
	t.Setenv("PJ_BLUETOOTH_DEVICE", "")
	svc := btconnect.NewBtConnectService(&fakeRunner{}, nil)
	err := svc.CheckAndConnect()
	require.Error(t, err)
	assert.ErrorIs(t, err, btconnect.ErrMissingDevice)
}

func TestCheckAndConnectConnects(t *testing.T) {
	t.Setenv("PJ_BLUETOOTH_DEVICE", "AA:BB:CC:DD:EE:FF")
	r := &fakeRunner{outputs: map[string]string{
		"bluetoothctl info AA:BB:CC:DD:EE:FF": "Connected: no\n",
	}}
	svc := btconnect.NewBtConnectService(r, nil)
	require.NoError(t, svc.CheckAndConnect())
	assert.Equal(t, []string{"bluetoothctl connect AA:BB:CC:DD:EE:FF"}, r.runs)
}

func TestRunMissingEnvFatal(t *testing.T) {
	t.Setenv("PJ_BLUETOOTH_DEVICE", "")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &recordingSleep{stop: 1, cancel: cancel}
	svc := btconnect.NewBtConnectService(
		&fakeRunner{}, newDiscardLogger(), btconnect.WithSleep(s.sleep),
	)
	err := svc.Run(ctx)
	require.ErrorIs(t, err, btconnect.ErrMissingDevice)
	assert.Empty(t, s.delays)
}

func TestRunContinuesOnConnectFailure(t *testing.T) {
	t.Setenv("PJ_BLUETOOTH_DEVICE", "AA:BB:CC:DD:EE:FF")
	runner := &fakeRunner{runErrs: map[string]error{
		"bluetoothctl connect AA:BB:CC:DD:EE:FF": errors.New("connect failed"),
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &recordingSleep{stop: 2, cancel: cancel}
	svc := btconnect.NewBtConnectService(
		runner, newDiscardLogger(), btconnect.WithSleep(s.sleep),
	)
	err := svc.Run(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Len(t, s.delays, 2)
	assert.Len(t, runner.runs, 2)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd system/soundsprout && go test ./btconnect/ -v`
Expected: compile FAIL — `svc.Run(ctx) wrong number of args`, `undefined: btconnect.WithSleep`

- [ ] **Step 3: Write minimal implementation**

In `system/soundsprout/btconnect/service.go`, replace the whole file with:

```go
package btconnect

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"aaronromeo.com/soundsprout/bluetooth"
)

const DefaultSleepTime = 15

// ErrMissingDevice reports that PJ_BLUETOOTH_DEVICE is not set. It is
// fatal: the environment is loaded once at process start, so retrying
// cannot help.
var ErrMissingDevice = errors.New("env var PJ_BLUETOOTH_DEVICE not set")

// SleepFunc pauses for d, aborting early when ctx is done.
type SleepFunc func(ctx context.Context, d time.Duration) error

type Service struct {
	runner bluetooth.Runner
	logger *slog.Logger
	sleep  SleepFunc
}

// Option configures a Service.
type Option func(*Service)

// WithSleep replaces the service's sleep implementation (test seam).
func WithSleep(fn SleepFunc) Option {
	return func(s *Service) { s.sleep = fn }
}

// NewBtConnectService returns a Bluetooth-connection maintenance service.
func NewBtConnectService(runner bluetooth.Runner, logger *slog.Logger, opts ...Option) *Service {
	s := &Service{runner: runner, logger: logger, sleep: sleepCtx}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Run maintains the connection until ctx is canceled. A missing
// PJ_BLUETOOTH_DEVICE is fatal; connect failures are logged and
// retried.
func (bt *Service) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := bt.CheckAndConnect(); err != nil {
			if errors.Is(err, ErrMissingDevice) {
				return err
			}
			bt.logger.Warn("bluetooth connect failed", "error", err)
		}
		if err := bt.sleep(ctx, DefaultSleepTime*time.Second); err != nil {
			return err
		}
	}
}

// CheckAndConnect performs one maintenance cycle for the configured device.
func (bt *Service) CheckAndConnect() error {
	device := os.Getenv("PJ_BLUETOOTH_DEVICE")
	if device == "" {
		return ErrMissingDevice
	}
	connected, err := bluetooth.IsConnected(bt.runner, device)
	if err != nil {
		return err
	}
	if !connected {
		return bluetooth.Connect(bt.runner, device)
	}
	return nil
}
```

In `system/soundsprout/cmd/main.go`:

Add `"context"` to the import block (first import, sorted with the stdlib group), and in the `btconnect` command's `Action` replace `err = connectService.Run()` with:

```go
					err = connectService.Run(context.Background())
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd system/soundsprout && go test ./btconnect/ -v`
Expected: PASS — all four tests (`TestCheckAndConnectMissingEnv`, `TestCheckAndConnectConnects`, `TestRunMissingEnvFatal`, `TestRunContinuesOnConnectFailure`)

Also run the full Go suite to catch the `main.go` change:

Run: `make test`
Expected: PASS (all packages, coverage written to `cover.out`)

- [ ] **Step 5: Commit**

```bash
git add system/soundsprout/btconnect/service.go system/soundsprout/btconnect/service_test.go system/soundsprout/cmd/main.go
git commit -m "fix(btconnect): keep running through connect failures"
```

---

### Task 3: Exponential backoff, reset on success, recovery log

**Files:**
- Modify: `system/soundsprout/btconnect/service.go` (`nextDelay` helper, failure counter in `Run`)
- Test: `system/soundsprout/btconnect/service_test.go` (extend `recordingSleep` with `onSleep`, add two tests)

**Interfaces:**
- Consumes: `SleepFunc`/`WithSleep`, `recordingSleep`, `newDiscardLogger` (Task 2).
- Produces: unexported `nextDelay(failures int) time.Duration` — covered indirectly through recorded sleep durations (no internal test package; per spec).

- [ ] **Step 1: Write the failing tests**

In `system/soundsprout/btconnect/service_test.go`:

Extend `recordingSleep` with an `onSleep` hook (replace the existing type and method):

```go
// recordingSleep records delays and cancels the loop's context once
// stop sleeps have been recorded. onSleep (optional) runs after each
// record, receiving the number of sleeps so far.
type recordingSleep struct {
	delays  []time.Duration
	stop    int
	cancel  context.CancelFunc
	onSleep func(n int)
}

func (r *recordingSleep) sleep(ctx context.Context, d time.Duration) error {
	r.delays = append(r.delays, d)
	if r.onSleep != nil {
		r.onSleep(len(r.delays))
	}
	if len(r.delays) >= r.stop {
		r.cancel()
	}
	return ctx.Err()
}
```

Add `bytes` to the import block (before `context`, alphabetically: `bytes`, `context`, `errors`, `io`, `log/slog`, `testing`, `time`).

Append these tests:

```go
func TestRunBackoffSequence(t *testing.T) {
	t.Setenv("PJ_BLUETOOTH_DEVICE", "AA:BB:CC:DD:EE:FF")
	runner := &fakeRunner{runErrs: map[string]error{
		"bluetoothctl connect AA:BB:CC:DD:EE:FF": errors.New("connect failed"),
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &recordingSleep{stop: 6, cancel: cancel}
	svc := btconnect.NewBtConnectService(
		runner, newDiscardLogger(), btconnect.WithSleep(s.sleep),
	)
	err := svc.Run(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []time.Duration{
		15 * time.Second,
		30 * time.Second,
		60 * time.Second,
		120 * time.Second,
		240 * time.Second,
		300 * time.Second,
	}, s.delays)
	assert.Len(t, runner.runs, 6)
}

func TestRunResetsBackoffOnSuccess(t *testing.T) {
	t.Setenv("PJ_BLUETOOTH_DEVICE", "AA:BB:CC:DD:EE:FF")
	const connectKey = "bluetoothctl connect AA:BB:CC:DD:EE:FF"
	runner := &fakeRunner{runErrs: map[string]error{
		connectKey: errors.New("connect failed"),
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	s := &recordingSleep{stop: 4, cancel: cancel, onSleep: func(n int) {
		switch n {
		case 2:
			delete(runner.runErrs, connectKey) // cycle 3 succeeds
		case 3:
			runner.runErrs[connectKey] = errors.New("connect failed") // cycle 4 fails
		}
	}}
	svc := btconnect.NewBtConnectService(runner, logger, btconnect.WithSleep(s.sleep))
	err := svc.Run(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []time.Duration{
		15 * time.Second,
		30 * time.Second,
		15 * time.Second,
		15 * time.Second,
	}, s.delays)
	assert.Contains(t, logs.String(), "bluetooth reconnected")
	assert.Contains(t, logs.String(), "failures=2")
}
```

Note: `TestRunContinuesOnConnectFailure` from Task 2 stays unchanged — it asserts no durations, so it remains valid.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd system/soundsprout && go test ./btconnect/ -v -run 'TestRunBackoffSequence|TestRunResetsBackoffOnSuccess'`
Expected: FAIL — both tests see `[15s 15s 15s ...]` instead of growing backoff (current fixed-interval `Run`).

- [ ] **Step 3: Write minimal implementation**

In `system/soundsprout/btconnect/service.go`:

Replace the const block:

```go
const (
	DefaultSleepTime = 15
	maxSleepTime     = 300
)
```

Replace `Run` and add `nextDelay` above it:

```go
// Run maintains the connection until ctx is canceled. A missing
// PJ_BLUETOOTH_DEVICE is fatal; connect failures are logged and
// retried with exponential backoff.
func (bt *Service) Run(ctx context.Context) error {
	failures := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := bt.CheckAndConnect()
		if err != nil {
			if errors.Is(err, ErrMissingDevice) {
				return err
			}
			failures++
			bt.logger.Warn(
				"bluetooth connect failed",
				"error", err,
				"failures", failures,
			)
		} else if failures > 0 {
			bt.logger.Info("bluetooth reconnected", "failures", failures)
			failures = 0
		}
		if sleepErr := bt.sleep(ctx, nextDelay(failures)); sleepErr != nil {
			return sleepErr
		}
	}
}

// nextDelay returns the backoff after n consecutive failures:
// 15s, 30s, 60s, 120s, 240s, then capped at 300s. n <= 0 returns the
// base interval.
func nextDelay(failures int) time.Duration {
	d := DefaultSleepTime * time.Second
	for i := 1; i < failures && d < maxSleepTime*time.Second; i++ {
		d *= 2
	}
	if d > maxSleepTime*time.Second {
		return maxSleepTime * time.Second
	}
	return d
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd system/soundsprout && go test ./btconnect/ -v`
Expected: PASS — all six tests.

- [ ] **Step 5: Lint**

Run: `golangci-lint run --config system/soundsprout/.golangci.yml ./...`
Workdir: `system/soundsprout/`
Expected: no issues.

- [ ] **Step 6: Commit**

```bash
git add system/soundsprout/btconnect/service.go system/soundsprout/btconnect/service_test.go
git commit -m "fix(btconnect): back off exponentially between failed connects"
```

---

### Task 4: Documentation updates

**Files:**
- Modify: `README.md:259`
- Modify: `docs/rfid-map-and-bluetooth.md:9` and `:140-144` (section 5.1)
- Modify: `AGENTS.md:36`

**Interfaces:**
- Consumes: behavior from Tasks 1–3 (backoff schedule 15s→300s, fatal missing env, self-heal).

- [ ] **Step 1: Update README.md**

In `README.md`, replace line 259:

```
It requires `PJ_BLUETOOTH_DEVICE` to be set (see step 16). Every 15s it checks `bluetoothctl info <device>` for `Connected: yes` and runs `bluetoothctl connect <device>` when disconnected.
```

with:

```
It requires `PJ_BLUETOOTH_DEVICE` to be set (see step 16). Every 15s it checks `bluetoothctl info <device>` for `Connected: yes` and runs `bluetoothctl connect <device>` when disconnected. Failed connect attempts are logged and retried with exponential backoff (15s doubling to a 300s cap, reset on success) instead of terminating the service, so the jukebox reconnects on its own when the speaker becomes available again.
```

- [ ] **Step 2: Update docs/rfid-map-and-bluetooth.md**

On line 9, replace:

```
with a Go service polling `bluetoothctl` every 15s to auto-reconnect.
```

with:

```
with a Go service polling `bluetoothctl` (15s cadence, backing off to at most 5 minutes after repeated failures) to auto-reconnect.
```

Replace the section 5.1 bullet list (lines 140–144):

```
- Runs an infinite loop every 15 seconds ([`service.go:47-70`](system/soundsprout/btconnect/service.go#L47-L70)).
- Reads `PJ_BLUETOOTH_DEVICE` env var (required) ([`service.go:48-51`](system/soundsprout/btconnect/service.go#L48-L51)).
- Checks connection: runs `bluetoothctl info <MAC>` ([`service.go:74`](system/soundsprout/btconnect/service.go#L74)).
- Parses output for `"Connected: yes"` lines ([`service.go:97-105`](system/soundsprout/btconnect/service.go#L97-L105)).
- If not connected: runs `bluetoothctl connect <MAC>` ([`service.go:85-94`](system/soundsprout/btconnect/service.go#L85-L94)).
```

with:

```
- Runs a context-aware loop (default cadence 15 seconds).
- Reads `PJ_BLUETOOTH_DEVICE` env var (required); a missing value is fatal (`ErrMissingDevice`) and stops the process.
- Checks connection: runs `bluetoothctl info <MAC>`, parses output for `"Connected: yes"`.
- If not connected: runs `bluetoothctl connect <MAC>`.
- Connect failures are logged (Warn) and retried with exponential backoff — 15s doubling to a 300s cap, reset on success — instead of terminating the service. A recovery is logged at Info. This prevents an absent speaker from FATALing the service under supervisor (the 2026-10-16 outage: fast exit/restart cycles wedged the Pi's Bluetooth stack; see the runbook in the incident handoff for radio-wedge recovery).
```

- [ ] **Step 3: Update AGENTS.md**

In `AGENTS.md`, on line 36, replace:

```
`btconnect` (maintains BT connection, loops every 15s, needs `PJ_BLUETOOTH_DEVICE`)
```

with:

```
`btconnect` (maintains BT connection, 15s loop with exponential backoff up to 300s on failures, needs `PJ_BLUETOOTH_DEVICE`)
```

- [ ] **Step 4: Verify docs build/lint**

Run: `pre-commit run --all-files`
Expected: PASS (trailing whitespace / EOF fixers / yaml checks clean on the edited files).

- [ ] **Step 5: Commit**

```bash
git add README.md docs/rfid-map-and-bluetooth.md AGENTS.md
git commit -m "docs: describe btconnect backoff and self-heal behavior"
```

---

### Task 5: Final verification

**Files:** none (verification only)

- [ ] **Step 1: Full test suite**

Run: `make test` (repo root)
Expected: PASS, coverage profile written.

- [ ] **Step 2: Go lint**

Run: `golangci-lint run --config system/soundsprout/.golangci.yml ./...`
Workdir: `system/soundsprout/`
Expected: no issues.

- [ ] **Step 3: Pre-commit**

Run: `pre-commit run --all-files`
Expected: PASS.

- [ ] **Step 4: Report**

Summarize: commits made, test/lint results, and remind the user that on-hardware validation (power the Esinkin off/on, watch `btconnect` stay RUNNING and auto-reconnect; `ssh pi@192.168.68.89`) is a post-deploy step requiring them, per spec §Verification. Deployment is push-to-main → `gitpull.sh` rebuilds and restarts the service.
