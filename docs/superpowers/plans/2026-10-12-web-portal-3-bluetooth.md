# Web Portal Part 3 — Bluetooth Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Manage the Bluetooth audio sink from the web portal — show status, scan, pair/trust/connect a device, switch the active output, and reconnect — and refactor `btconnect` onto one shared `bluetoothctl` wrapper.

**Architecture:** A new `bluetooth` package wraps `bluetoothctl` behind a `Runner` seam and holds all output parsing in pure functions. `btconnect` and the portal both use it. The portal adds `Settings` (the active MAC in `/home/pi/.soundsprout/conf`), `AudioConfig` (renders `.asoundrc` + cmus `rc` from the existing `templategen` templates and runs a whitelisted restart pipeline), and a `Bluetooth` façade.

**Tech Stack:** Go stdlib, existing `urfave/cli`, `testify`; Supervisor + BlueZ (`bluetoothctl`) + `alsactl`; a narrow `NOPASSWD` sudoers whitelist.

**Depends on:** `docs/superpowers/plans/2026-10-12-web-portal-1-framework.md` and `...-2-cards.md`.

## Global Constraints

- Go module path: `aaronromeo.com/soundsprout`. Go floor: 1.21 (CI).
- No shell anywhere: always `exec.Command(name, args...)` / `exec.CommandContext`. Never pass user input to a shell.
- Go tests live in external `_test` packages (e.g. `package portal_test`) — the `testpackage` linter is enabled in `system/soundsprout/.golangci.yml`.
- Canonical paths (used as defaults; tests inject temp paths):
  - settings dotenv `/home/pi/.soundsprout/conf`, settings lock `/tmp/pyrfid_jukebox.settings.lock`
  - `.asoundrc` `/home/pi/.asoundrc`, cmus rc `/home/pi/.config/cmus/rc`
- Strict MAC regex `^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`; no user input reaches a shell or an unchecked path.
- Apply pipeline (in order): write `~/.asoundrc` and `~/.config/cmus/rc`, then `sudo supervisorctl restart btconnect`, `sudo supervisorctl restart cmus_manager`, `sudo alsactl restore`.
- No authentication (LAN-only) — POSTs require same-host `Origin`/`Referer`.
- Run Go tests with `ASDF_GOLANG_VERSION=1.25.6` locally.
- Python: black line-length 79, flake8 max-line-length 80 (no Python in this plan).

---

### Task 1: Shared `bluetooth` package

**Files:**
- Create: `system/soundsprout/bluetooth/runner.go`
- Create: `system/soundsprout/bluetooth/bluetooth.go`
- Test: `system/soundsprout/bluetooth/bluetooth_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `type Runner interface { Output(name string, args ...string) (string, error); Run(name string, args ...string) error }`
  - `type OSRunner struct{}` implementing `Runner`.
  - `type Device struct { MAC, Name string }`
  - `type Info struct { MAC, Name string; Paired, Trusted, Connected bool }`
  - `func ValidMAC(mac string) bool`
  - `func ParseInfo(out string) Info`
  - `func ParseDevices(out string) []Device`
  - `func Status(r Runner, mac string) (Info, error)`
  - `func IsConnected(r Runner, mac string) (bool, error)`
  - `func Connect(r Runner, mac string) error`
  - `func Reconnect(r Runner, mac string) error`
  - `func KnownDevices(r Runner) ([]Device, error)`
  - `func Scan(r Runner, timeout time.Duration) ([]Device, error)`
  - `func Pair(r Runner, mac string) error`

- [ ] **Step 1: Write the failing tests**

Create `system/soundsprout/bluetooth/bluetooth_test.go`:

```go
package bluetooth_test

import (
	"testing"
	"time"

	"aaronromeo.com/soundsprout/bluetooth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRunner struct {
	outputs map[string]string
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
	return nil
}

func TestParseInfo(t *testing.T) {
	out := "Device AA:BB:CC:DD:EE:FF\n\tName: Kitchen Speaker\n" +
		"\tPaired: yes\n\tTrusted: yes\n\tConnected: yes\n"
	info := bluetooth.ParseInfo(out)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", info.MAC)
	assert.Equal(t, "Kitchen Speaker", info.Name)
	assert.True(t, info.Paired)
	assert.True(t, info.Trusted)
	assert.True(t, info.Connected)
}

func TestParseDevices(t *testing.T) {
	out := "Device AA:BB:CC:DD:EE:FF Kitchen Speaker\n" +
		"Device 11:22:33:44:55:66\n"
	devices := bluetooth.ParseDevices(out)
	require.Len(t, devices, 2)
	assert.Equal(t, "11:22:33:44:55:66", devices[0].MAC)
	assert.Equal(t, "Kitchen Speaker", devices[1].Name)
}

func TestValidMAC(t *testing.T) {
	assert.True(t, bluetooth.ValidMAC("AA:BB:CC:DD:EE:FF"))
	assert.False(t, bluetooth.ValidMAC("not-a-mac"))
}

func TestReconnectSkipsWhenConnected(t *testing.T) {
	r := &fakeRunner{outputs: map[string]string{
		"bluetoothctl info AA:BB:CC:DD:EE:FF": "Connected: yes\n",
	}}
	require.NoError(t, bluetooth.Reconnect(r, "AA:BB:CC:DD:EE:FF"))
	assert.Empty(t, r.runs)
}

func TestReconnectConnectsWhenNotConnected(t *testing.T) {
	r := &fakeRunner{outputs: map[string]string{
		"bluetoothctl info AA:BB:CC:DD:EE:FF": "Connected: no\n",
	}}
	require.NoError(t, bluetooth.Reconnect(r, "AA:BB:CC:DD:EE:FF"))
	assert.Equal(t, []string{"bluetoothctl connect AA:BB:CC:DD:EE:FF"}, r.runs)
}

func TestPairPairsTrustsConnects(t *testing.T) {
	r := &fakeRunner{outputs: map[string]string{}}
	require.NoError(t, bluetooth.Pair(r, "AA:BB:CC:DD:EE:FF"))
	assert.Equal(t, []string{
		"bluetoothctl pair AA:BB:CC:DD:EE:FF",
		"bluetoothctl trust AA:BB:CC:DD:EE:FF",
		"bluetoothctl connect AA:BB:CC:DD:EE:FF",
	}, r.runs)
}

func TestScanReturnsKnownDevices(t *testing.T) {
	r := &fakeRunner{outputs: map[string]string{
		"bluetoothctl devices": "Device AA:BB:CC:DD:EE:FF Speaker\n",
	}}
	devices, err := bluetooth.Scan(r, 3*time.Second)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", devices[0].MAC)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./bluetooth/...`
Expected: FAIL — package/types not defined.

- [ ] **Step 3: Write `runner.go`**

Create `system/soundsprout/bluetooth/runner.go`:

```go
package bluetooth

import (
	"context"
	"os/exec"
	"time"
)

const commandTimeout = 15 * time.Second

// Runner is the single impure seam for external commands.
type Runner interface {
	Output(name string, args ...string) (string, error)
	Run(name string, args ...string) error
}

type OSRunner struct{}

func (OSRunner) Output(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	//nolint:gosec // name/args come from fixed internal callers, never user input.
	out, err := exec.CommandContext(ctx, name, args...).Output()
	return string(out), err
}

func (OSRunner) Run(name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	//nolint:gosec // name/args come from fixed internal callers, never user input.
	return exec.CommandContext(ctx, name, args...).Run()
}
```

- [ ] **Step 4: Write `bluetooth.go`**

Create `system/soundsprout/bluetooth/bluetooth.go`:

```go
package bluetooth

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	bluetoothctlCommand = "bluetoothctl"
	yesValue            = "yes"
)

// ValidMAC reports whether mac is a colon-separated 48-bit address.
func ValidMAC(mac string) bool {
	re := regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`)
	return re.MatchString(mac)
}

type Device struct {
	MAC  string
	Name string
}

type Info struct {
	MAC       string
	Name      string
	Paired    bool
	Trusted   bool
	Connected bool
}

// ParseInfo parses the output of `bluetoothctl info <mac>`.
func ParseInfo(out string) Info {
	info := Info{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Device "):
			if fields := strings.Fields(line); len(fields) >= 2 {
				info.MAC = fields[1]
			}
		case strings.HasPrefix(line, "Name:"):
			info.Name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
		case strings.HasPrefix(line, "Paired:"):
			info.Paired = strings.Contains(line, yesValue)
		case strings.HasPrefix(line, "Trusted:"):
			info.Trusted = strings.Contains(line, yesValue)
		case strings.HasPrefix(line, "Connected:"):
			info.Connected = strings.Contains(line, yesValue)
		}
	}
	return info
}

// ParseDevices parses `bluetoothctl devices` lines, sorted by MAC.
func ParseDevices(out string) []Device {
	var devices []Device
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 2 && fields[0] == "Device" {
			name := ""
			if len(fields) > 2 {
				name = strings.Join(fields[2:], " ")
			}
			devices = append(devices, Device{MAC: fields[1], Name: name})
		}
	}
	sort.Slice(devices, func(i, j int) bool { return devices[i].MAC < devices[j].MAC })
	return devices
}

// Status returns the parsed info for mac.
func Status(r Runner, mac string) (Info, error) {
	out, err := r.Output(bluetoothctlCommand, "info", mac)
	if err != nil {
		return Info{}, fmt.Errorf("bluetoothctl info %s: %w", mac, err)
	}
	return ParseInfo(out), nil
}

// IsConnected reports whether mac is currently connected.
func IsConnected(r Runner, mac string) (bool, error) {
	info, err := Status(r, mac)
	if err != nil {
		return false, err
	}
	return info.Connected, nil
}

// Connect connects to an already-paired mac.
func Connect(r Runner, mac string) error {
	if err := r.Run(bluetoothctlCommand, "connect", mac); err != nil {
		return fmt.Errorf("bluetoothctl connect %s: %w", mac, err)
	}
	return nil
}

// Reconnect connects to mac only if it is not already connected.
func Reconnect(r Runner, mac string) error {
	connected, err := IsConnected(r, mac)
	if err != nil {
		return err
	}
	if connected {
		return nil
	}
	return Connect(r, mac)
}

// KnownDevices lists the devices bluetoothctl already knows about.
func KnownDevices(r Runner) ([]Device, error) {
	out, err := r.Output(bluetoothctlCommand, "devices")
	if err != nil {
		return nil, fmt.Errorf("bluetoothctl devices: %w", err)
	}
	return ParseDevices(out), nil
}

// Scan runs a timed scan, then returns the known devices.
func Scan(r Runner, timeout time.Duration) ([]Device, error) {
	secs := int(timeout.Seconds())
	if secs < 1 {
		secs = 1
	}
	if _, err := r.Output(bluetoothctlCommand, "--timeout", fmt.Sprintf("%d", secs), "scan", "on"); err != nil {
		return nil, fmt.Errorf("bluetoothctl scan: %w", err)
	}
	return KnownDevices(r)
}

// Pair pairs, trusts, and connects mac.
func Pair(r Runner, mac string) error {
	if !ValidMAC(mac) {
		return fmt.Errorf("invalid MAC address %q", mac)
	}
	if err := r.Run(bluetoothctlCommand, "pair", mac); err != nil {
		return fmt.Errorf("pair %s: %w", mac, err)
	}
	if err := r.Run(bluetoothctlCommand, "trust", mac); err != nil {
		return fmt.Errorf("trust %s: %w", mac, err)
	}
	return Connect(r, mac)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./bluetooth/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add system/soundsprout/bluetooth
git commit -m "feat(bluetooth): add shared bluetoothctl wrapper"
```

---

### Task 2: Refactor `btconnect` onto the shared package

**Files:**
- Modify: `system/soundsprout/btconnect/service.go` (replace entire file)
- Test: `system/soundsprout/btconnect/service_test.go` (create)
- Modify: `system/soundsprout/cmd/main.go` (wiring only)

**Interfaces:**
- Consumes: `bluetooth.Runner`, `bluetooth.IsConnected`, `bluetooth.Connect`.
- Produces: `func NewBtConnectService(runner bluetooth.Runner, logger *slog.Logger) *Service`; `func (bt *Service) CheckAndConnect() error`; `func (bt *Service) Run() error`.

- [ ] **Step 1: Write the failing test**

Create `system/soundsprout/btconnect/service_test.go`:

```go
package btconnect_test

import (
	"testing"

	"aaronromeo.com/soundsprout/btconnect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRunner struct {
	outputs map[string]string
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
	return nil
}

func TestCheckAndConnectMissingEnv(t *testing.T) {
	t.Setenv("PJ_BLUETOOTH_DEVICE", "")
	svc := btconnect.NewBtConnectService(&fakeRunner{}, nil)
	require.Error(t, svc.CheckAndConnect())
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./btconnect/...`
Expected: FAIL — signature/type mismatch.

- [ ] **Step 3: Replace `service.go`**

Replace the entire contents of `system/soundsprout/btconnect/service.go` with:

```go
package btconnect

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"aaronromeo.com/soundsprout/bluetooth"
)

const DefaultSleepTime = 15

type Service struct {
	runner bluetooth.Runner
	logger *slog.Logger
}

// NewBtConnectService returns a Bluetooth-connection maintenance service.
func NewBtConnectService(runner bluetooth.Runner, logger *slog.Logger) *Service {
	return &Service{runner: runner, logger: logger}
}

func (bt *Service) Run() error {
	for {
		if err := bt.CheckAndConnect(); err != nil {
			return err
		}
		time.Sleep(DefaultSleepTime * time.Second)
	}
}

// CheckAndConnect performs one maintenance cycle for the configured device.
func (bt *Service) CheckAndConnect() error {
	device := os.Getenv("PJ_BLUETOOTH_DEVICE")
	if device == "" {
		return fmt.Errorf("env var PJ_BLUETOOTH_DEVICE not set")
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

- [ ] **Step 4: Update `cmd/main.go` btconnect wiring**

In `system/soundsprout/cmd/main.go`, add `"aaronromeo.com/soundsprout/bluetooth"` to the
import block and replace the `btconnect` command action body with:

```go
				Action: func(c *cli.Context) error {
					connectService := btconnect.NewBtConnectService(
						bluetooth.OSRunner{},
						logger,
					)
					err = connectService.Run()
					if err != nil {
						logger.Error(
							"btconnect failure",
							"error", err,
						)
					}
					return nil
				},
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./btconnect/... ./templategen/... ./bluetooth/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add system/soundsprout/btconnect system/soundsprout/cmd/main.go
git commit -m "refactor(btconnect): reuse shared bluetooth package"
```

---

### Task 3: `portal.Settings`

**Files:**
- Create: `system/soundsprout/portal/settings.go`
- Test: `system/soundsprout/portal/settings_test.go`

**Interfaces:**
- Consumes: `writeFileAtomic` (from `portal/fileutil.go`, added in Part 2).
- Produces:
  - `func NewSettings(path, lockPath string) *Settings`
  - `func (s *Settings) Get(key string) (string, error)`
  - `func (s *Settings) Set(key, value string) error`

- [ ] **Step 1: Write the failing tests**

Create `system/soundsprout/portal/settings_test.go`:

```go
package portal_test

import (
	"os"
	"path/filepath"
	"testing"

	"aaronromeo.com/soundsprout/portal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSettings(t *testing.T) (*portal.Settings, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "conf")
	return portal.NewSettings(path, filepath.Join(dir, "settings.lock")), path
}

func TestSettingsGetMissing(t *testing.T) {
	s, _ := newTestSettings(t)
	v, err := s.Get("PJ_BLUETOOTH_DEVICE")
	require.NoError(t, err)
	assert.Equal(t, "", v)
}

func TestSettingsSetAndGet(t *testing.T) {
	s, _ := newTestSettings(t)
	require.NoError(t, s.Set("PJ_BLUETOOTH_DEVICE", "AA:BB:CC:DD:EE:FF"))
	v, err := s.Get("PJ_BLUETOOTH_DEVICE")
	require.NoError(t, err)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", v)
}

func TestSettingsSetPreservesOtherKeys(t *testing.T) {
	s, path := newTestSettings(t)
	require.NoError(t, os.WriteFile(path, []byte("PJ_DEFAULT_BRANCH=main\n"), 0o644))
	require.NoError(t, s.Set("PJ_BLUETOOTH_DEVICE", "AA:BB:CC:DD:EE:FF"))
	branch, err := s.Get("PJ_DEFAULT_BRANCH")
	require.NoError(t, err)
	assert.Equal(t, "main", branch)
	device, err := s.Get("PJ_BLUETOOTH_DEVICE")
	require.NoError(t, err)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", device)
}

func TestSettingsSetReplacesExistingLine(t *testing.T) {
	s, path := newTestSettings(t)
	require.NoError(t, os.WriteFile(path, []byte("PJ_BLUETOOTH_DEVICE=old\n"), 0o644))
	require.NoError(t, s.Set("PJ_BLUETOOTH_DEVICE", "new"))
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "PJ_BLUETOOTH_DEVICE=new\n", string(raw))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./portal/... -run TestSettings`
Expected: FAIL — `undefined: portal.NewSettings`.

- [ ] **Step 3: Write `settings.go`**

Create `system/soundsprout/portal/settings.go`:

```go
package portal

import (
	"os"
	"strings"
	"syscall"
)

type Settings struct {
	path     string
	lockPath string
}

// NewSettings returns a dotenv store at path, locked via lockPath for writes.
func NewSettings(path, lockPath string) *Settings {
	return &Settings{path: path, lockPath: lockPath}
}

// Get returns the value for key, or "" if absent.
func (s *Settings) Get(key string) (string, error) {
	values, err := s.read()
	if err != nil {
		return "", err
	}
	return values[key], nil
}

// Set updates one dotenv key in place, preserving other keys and comments.
func (s *Settings) Set(key, value string) error {
	lock, err := os.OpenFile(s.lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer func() { _ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN) }()

	lines, err := s.readLines()
	if err != nil {
		return err
	}
	replaced := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), key+"=") {
			lines[i] = key + "=" + value
			replaced = true
		}
	}
	if !replaced {
		lines = append(lines, key+"="+value)
	}
	content := strings.Join(lines, "\n") + "\n"
	return writeFileAtomic(s.path, content)
}

func (s *Settings) read() (map[string]string, error) {
	values := map[string]string{}
	lines, err := s.readLines()
	if err != nil {
		return nil, err
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if found {
			values[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return values, nil
}

func (s *Settings) readLines() ([]string, error) {
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	content := strings.TrimRight(string(raw), "\n")
	if content == "" {
		return nil, nil
	}
	return strings.Split(content, "\n"), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./portal/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add system/soundsprout/portal/settings.go system/soundsprout/portal/settings_test.go
git commit -m "feat(portal): add locked dotenv settings store"
```

---

### Task 4: `portal.AudioConfig`

**Files:**
- Create: `system/soundsprout/portal/config.go`
- Create: `system/soundsprout/portal/fake_runner_test.go` (shared test helper)
- Test: `system/soundsprout/portal/config_test.go`

**Interfaces:**
- Consumes: `templategen.GenerateTemplate`, `bluetooth.Runner`, `writeFileAtomic`.
- Produces:
  - `func NewAudioConfig(asoundrcPath, cmusRcPath string, runner bluetooth.Runner) *AudioConfig`
  - `func (a *AudioConfig) Apply(mac string) error`

- [ ] **Step 1: Create the shared fake runner test helper**

Create `system/soundsprout/portal/fake_runner_test.go`:

```go
package portal_test

type fakeRunner struct {
	outputs map[string]string
	runs    []string
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{outputs: map[string]string{}}
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
	return nil
}
```

- [ ] **Step 2: Write the failing test**

Create `system/soundsprout/portal/config_test.go`:

```go
package portal_test

import (
	"os"
	"path/filepath"
	"testing"

	"aaronromeo.com/soundsprout/portal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAudioConfigApplyWritesAndRestarts(t *testing.T) {
	dir := t.TempDir()
	asoundrc := filepath.Join(dir, ".asoundrc")
	cmusrc := filepath.Join(dir, "cmus-rc")
	r := newFakeRunner()
	audio := portal.NewAudioConfig(asoundrc, cmusrc, r)

	require.NoError(t, audio.Apply("AA:BB:CC:DD:EE:FF"))

	asound, err := os.ReadFile(asoundrc)
	require.NoError(t, err)
	assert.Contains(t, string(asound), `device "AA:BB:CC:DD:EE:FF"`)

	cmus, err := os.ReadFile(cmusrc)
	require.NoError(t, err)
	assert.Contains(t, string(cmus), "bluealsa:DEV=AA:BB:CC:DD:EE:FF")

	assert.Equal(t, []string{
		"sudo supervisorctl restart btconnect",
		"sudo supervisorctl restart cmus_manager",
		"sudo alsactl restore",
	}, r.runs)
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./portal/... -run TestAudioConfig`
Expected: FAIL — `undefined: portal.NewAudioConfig`.

- [ ] **Step 4: Write `config.go`**

Create `system/soundsprout/portal/config.go`:

```go
package portal

import (
	"fmt"

	"aaronromeo.com/soundsprout/bluetooth"
	"aaronromeo.com/soundsprout/templategen"
)

const sudoCommand = "sudo"

type AudioConfig struct {
	asoundrcPath string
	cmusRcPath   string
	runner       bluetooth.Runner
}

// NewAudioConfig returns an AudioConfig that renders and applies audio configs.
func NewAudioConfig(asoundrcPath, cmusRcPath string, runner bluetooth.Runner) *AudioConfig {
	return &AudioConfig{
		asoundrcPath: asoundrcPath,
		cmusRcPath:   cmusRcPath,
		runner:       runner,
	}
}

// Apply renders the templated audio configs for mac, writes them, and restarts
// the services that consume them.
func (a *AudioConfig) Apply(mac string) error {
	subs := map[string]string{"PJ_BLUETOOTH_DEVICE": mac}

	asoundrc, err := templategen.GenerateTemplate("templates/asoundrc.txt", subs)
	if err != nil {
		return fmt.Errorf("render asoundrc: %w", err)
	}
	cmusrc, err := templategen.GenerateTemplate("templates/config-cmus-rc.txt", subs)
	if err != nil {
		return fmt.Errorf("render cmus rc: %w", err)
	}
	if err := writeFileAtomic(a.asoundrcPath, asoundrc); err != nil {
		return err
	}
	if err := writeFileAtomic(a.cmusRcPath, cmusrc); err != nil {
		return err
	}

	commands := [][]string{
		{sudoCommand, "supervisorctl", "restart", "btconnect"},
		{sudoCommand, "supervisorctl", "restart", "cmus_manager"},
		{sudoCommand, "alsactl", "restore"},
	}
	for _, c := range commands {
		if err := a.runner.Run(c[0], c[1:]...); err != nil {
			return fmt.Errorf("run %v: %w", c, err)
		}
	}
	return nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./portal/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add system/soundsprout/portal/config.go system/soundsprout/portal/config_test.go \
  system/soundsprout/portal/fake_runner_test.go
git commit -m "feat(portal): apply templated audio configs"
```

---

### Task 5: `portal.Bluetooth` façade

**Files:**
- Create: `system/soundsprout/portal/bluetooth.go`
- Test: `system/soundsprout/portal/bluetooth_test.go`

**Interfaces:**
- Consumes: `bluetooth.*`, `Settings`, `AudioConfig`.
- Produces:
  - `func NewBluetooth(runner bluetooth.Runner, settings *Settings, audio *AudioConfig) *Bluetooth`
  - `func (b *Bluetooth) ActiveMAC() (string, error)`
  - `func (b *Bluetooth) Status(mac string) (bluetooth.Info, error)`
  - `func (b *Bluetooth) KnownDevices() ([]bluetooth.Device, error)`
  - `func (b *Bluetooth) Scan(timeout time.Duration) ([]bluetooth.Device, error)`
  - `func (b *Bluetooth) Activate(mac string) error`
  - `func (b *Bluetooth) PairAndActivate(mac string) error`
  - `func (b *Bluetooth) Reconnect() error`

- [ ] **Step 1: Write the failing test**

Create `system/soundsprout/portal/bluetooth_test.go`:

```go
package portal_test

import (
	"path/filepath"
	"testing"

	"aaronromeo.com/soundsprout/portal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestBluetooth(t *testing.T) (*portal.Bluetooth, *fakeRunner, *portal.Settings) {
	t.Helper()
	dir := t.TempDir()
	r := newFakeRunner()
	settings := portal.NewSettings(
		filepath.Join(dir, "conf"),
		filepath.Join(dir, "settings.lock"),
	)
	audio := portal.NewAudioConfig(
		filepath.Join(dir, ".asoundrc"),
		filepath.Join(dir, "cmus-rc"),
		r,
	)
	return portal.NewBluetooth(r, settings, audio), r, settings
}

func TestBluetoothActivatePersistsAndApplies(t *testing.T) {
	bt, r, settings := newTestBluetooth(t)
	require.NoError(t, bt.Activate("AA:BB:CC:DD:EE:FF"))

	mac, err := settings.Get("PJ_BLUETOOTH_DEVICE")
	require.NoError(t, err)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", mac)

	assert.Contains(t, r.runs, "bluetoothctl connect AA:BB:CC:DD:EE:FF")
	assert.Contains(t, r.runs, "sudo supervisorctl restart btconnect")
}

func TestBluetoothActivateRejectsBadMAC(t *testing.T) {
	bt, _, _ := newTestBluetooth(t)
	require.Error(t, bt.Activate("nope"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./portal/... -run TestBluetooth`
Expected: FAIL — `undefined: portal.NewBluetooth`.

- [ ] **Step 3: Write `bluetooth.go`**

Create `system/soundsprout/portal/bluetooth.go`:

```go
package portal

import (
	"fmt"
	"time"

	"aaronromeo.com/soundsprout/bluetooth"
)

const bluetoothDeviceKey = "PJ_BLUETOOTH_DEVICE"

type Bluetooth struct {
	runner   bluetooth.Runner
	settings *Settings
	audio    *AudioConfig
}

// NewBluetooth returns a façade over bluetoothctl plus config apply.
func NewBluetooth(runner bluetooth.Runner, settings *Settings, audio *AudioConfig) *Bluetooth {
	return &Bluetooth{runner: runner, settings: settings, audio: audio}
}

// ActiveMAC returns the configured device MAC, or "" if none.
func (b *Bluetooth) ActiveMAC() (string, error) {
	return b.settings.Get(bluetoothDeviceKey)
}

// Status returns the parsed status for mac.
func (b *Bluetooth) Status(mac string) (bluetooth.Info, error) {
	return bluetooth.Status(b.runner, mac)
}

// KnownDevices lists devices bluetoothctl already knows about.
func (b *Bluetooth) KnownDevices() ([]bluetooth.Device, error) {
	return bluetooth.KnownDevices(b.runner)
}

// Scan runs a timed scan and returns the known devices.
func (b *Bluetooth) Scan(timeout time.Duration) ([]bluetooth.Device, error) {
	return bluetooth.Scan(b.runner, timeout)
}

// Activate makes an already-paired device the active audio output.
func (b *Bluetooth) Activate(mac string) error {
	if !bluetooth.ValidMAC(mac) {
		return fmt.Errorf("invalid MAC address %q", mac)
	}
	if err := bluetooth.Connect(b.runner, mac); err != nil {
		return err
	}
	return b.persistAndApply(mac)
}

// PairAndActivate pairs, trusts, and connects a new device, then applies it.
func (b *Bluetooth) PairAndActivate(mac string) error {
	if !bluetooth.ValidMAC(mac) {
		return fmt.Errorf("invalid MAC address %q", mac)
	}
	if err := bluetooth.Pair(b.runner, mac); err != nil {
		return err
	}
	return b.persistAndApply(mac)
}

// Reconnect reconnects the currently configured device.
func (b *Bluetooth) Reconnect() error {
	mac, err := b.ActiveMAC()
	if err != nil {
		return err
	}
	if mac == "" {
		return fmt.Errorf("no active bluetooth device configured")
	}
	return bluetooth.Reconnect(b.runner, mac)
}

func (b *Bluetooth) persistAndApply(mac string) error {
	if err := b.settings.Set(bluetoothDeviceKey, mac); err != nil {
		return err
	}
	return b.audio.Apply(mac)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./portal/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add system/soundsprout/portal/bluetooth.go system/soundsprout/portal/bluetooth_test.go
git commit -m "feat(portal): add bluetooth façade"
```

---

### Task 6: Bluetooth HTTP handlers, template, and wiring

**Files:**
- Modify: `system/soundsprout/portal/server.go` (replace entire file)
- Modify: `system/soundsprout/portal/service.go` (replace entire file)
- Create: `system/soundsprout/portal/templates/bluetooth.html`
- Modify: `system/soundsprout/portal/templates/cards.html` (add nav link)
- Modify: `system/soundsprout/portal/server_test.go` (update helper, add test)

**Interfaces:**
- Consumes: `MapStore`, `Library`, `Bluetooth`.
- Produces:
  - `func NewServer(mapStore *MapStore, library *Library, bt *Bluetooth, logger *slog.Logger) (*Server, error)`
  - `func NewService(logger *slog.Logger) (*Server, error)` (signature unchanged)
  - Routes: `GET /bluetooth`, `POST /bluetooth/scan`, `POST /bluetooth/pair`, `POST /bluetooth/connect`, `POST /bluetooth/reconnect`.

- [ ] **Step 1: Replace `server.go`**

Replace the entire contents of `system/soundsprout/portal/server.go` with:

```go
package portal

import (
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"
)

//go:embed templates/*.html
var templatesFS embed.FS

// scanTimeout bounds bluetoothctl discovery.
const scanTimeout = 5 * time.Second

type cardRow struct {
	ID       string
	Path     string
	Exists   bool
	MP3Count int
}

type Server struct {
	mapStore *MapStore
	library  *Library
	bt       *Bluetooth
	logger   *slog.Logger
	tmpl     *template.Template
}

// NewServer parses the embedded templates and returns a Server.
func NewServer(mapStore *MapStore, library *Library, bt *Bluetooth, logger *slog.Logger) (*Server, error) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Server{mapStore: mapStore, library: library, bt: bt, logger: logger, tmpl: tmpl}, nil
}

// Routes returns the portal's HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/cards", s.handleCards)
	mux.HandleFunc("/cards/assign", s.handleAssign)
	mux.HandleFunc("/cards/remove", s.handleRemove)
	mux.HandleFunc("/bluetooth", s.handleBluetooth)
	mux.HandleFunc("/bluetooth/scan", s.handleScan)
	mux.HandleFunc("/bluetooth/pair", s.handlePair)
	mux.HandleFunc("/bluetooth/connect", s.handleConnect)
	mux.HandleFunc("/bluetooth/reconnect", s.handleReconnect)
	return s.csrfGuard(mux)
}

// csrfGuard rejects POSTs whose Origin/Referer host differs from the request host.
func (s *Server) csrfGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			origin := r.Header.Get("Origin")
			if origin == "" {
				origin = r.Header.Get("Referer")
			}
			if origin != "" {
				if u, err := url.Parse(origin); err != nil || u.Host != r.Host {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil && s.logger != nil {
		s.logger.Error("render", "template", name, "error", err)
	}
}

func (s *Server) redirect(w http.ResponseWriter, r *http.Request) {
	target := r.Referer()
	if target == "" {
		target = "/cards"
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.render(w, "index.html", map[string]any{"Title": "SoundSprout"})
}

func (s *Server) handleCards(w http.ResponseWriter, r *http.Request) {
	data, err := s.mapStore.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	folders, err := s.library.Folders()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rows := make([]cardRow, 0, len(data))
	unassigned := []string{}
	for id, path := range data {
		if path == "" {
			unassigned = append(unassigned, id)
			continue
		}
		info, statErr := os.Stat(path)
		exists := statErr == nil && info.IsDir()
		count := 0
		if exists {
			count, _ = countMP3(path)
		}
		rows = append(rows, cardRow{ID: id, Path: path, Exists: exists, MP3Count: count})
	}
	s.render(w, "cards.html", map[string]any{
		"Rows":       rows,
		"Unassigned": unassigned,
		"Folders":    folders,
	})
}

func (s *Server) handleAssign(w http.ResponseWriter, r *http.Request) {
	cardID := r.FormValue("card_id")
	folder := r.FormValue("folder")

	data, err := s.mapStore.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, ok := data[cardID]; !ok {
		http.Error(w, "unknown card id", http.StatusBadRequest)
		return
	}
	folderPath, err := s.library.Resolve(folder)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.mapStore.Assign(cardID, folderPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.redirect(w, r)
}

func (s *Server) handleRemove(w http.ResponseWriter, r *http.Request) {
	if err := s.mapStore.Remove(r.FormValue("card_id")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.redirect(w, r)
}

func (s *Server) handleBluetooth(w http.ResponseWriter, r *http.Request) {
	s.renderBluetooth(w, nil)
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	devices, err := s.bt.Scan(scanTimeout)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	s.renderBluetooth(w, devices)
}

func (s *Server) handlePair(w http.ResponseWriter, r *http.Request) {
	if err := s.bt.PairAndActivate(r.FormValue("mac")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.redirect(w, r)
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	if err := s.bt.Activate(r.FormValue("mac")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.redirect(w, r)
}

func (s *Server) handleReconnect(w http.ResponseWriter, r *http.Request) {
	if err := s.bt.Reconnect(); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	s.redirect(w, r)
}

func (s *Server) renderBluetooth(w http.ResponseWriter, discovered any) {
	mac, err := s.bt.ActiveMAC()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var status any
	if mac != "" {
		if st, stErr := s.bt.Status(mac); stErr == nil {
			status = st
		}
	}
	known, _ := s.bt.KnownDevices()
	s.render(w, "bluetooth.html", map[string]any{
		"ActiveMAC":  mac,
		"Status":     status,
		"Known":      known,
		"Discovered": discovered,
	})
}
```

- [ ] **Step 2: Replace `service.go`**

Replace the entire contents of `system/soundsprout/portal/service.go` with:

```go
package portal

import (
	"log/slog"

	"aaronromeo.com/soundsprout/bluetooth"
)

const (
	DefaultMusicDir     = "/home/pi/workspace/pyrfid-jukebox/music"
	DefaultMapFile      = "/home/pi/workspace/pyrfid-jukebox/server/rfid_map.json"
	DefaultMapLock      = "/tmp/pyrfid_jukebox.map.lock"
	DefaultSettingsFile = "/home/pi/.soundsprout/conf"
	DefaultSettingsLock = "/tmp/pyrfid_jukebox.settings.lock"
	DefaultAsoundrc     = "/home/pi/.asoundrc"
	DefaultCmusRc       = "/home/pi/.config/cmus/rc"
)

// Config holds every injectable path the portal uses.
type Config struct {
	MusicDir     string
	MapFile      string
	MapLock      string
	SettingsFile string
	SettingsLock string
	AsoundrcPath string
	CmusRcPath   string
}

// DefaultConfig returns the canonical on-device paths.
func DefaultConfig() Config {
	return Config{
		MusicDir:     DefaultMusicDir,
		MapFile:      DefaultMapFile,
		MapLock:      DefaultMapLock,
		SettingsFile: DefaultSettingsFile,
		SettingsLock: DefaultSettingsLock,
		AsoundrcPath: DefaultAsoundrc,
		CmusRcPath:   DefaultCmusRc,
	}
}

// NewService wires the portal's dependencies and returns a ready Server.
func NewService(logger *slog.Logger) (*Server, error) {
	cfg := DefaultConfig()
	runner := bluetooth.OSRunner{}
	mapStore := NewMapStore(cfg.MapFile, cfg.MapLock)
	library := NewLibrary(cfg.MusicDir)
	settings := NewSettings(cfg.SettingsFile, cfg.SettingsLock)
	audio := NewAudioConfig(cfg.AsoundrcPath, cfg.CmusRcPath, runner)
	bt := NewBluetooth(runner, settings, audio)
	return NewServer(mapStore, library, bt, logger)
}
```

- [ ] **Step 3: Create `bluetooth.html`**

Create `system/soundsprout/portal/templates/bluetooth.html`:

```html
<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Bluetooth</title></head>
<body>
<h1>Bluetooth</h1>
<p><a href="/">Home</a> · <a href="/cards">Cards</a></p>

<h2>Status</h2>
{{if .ActiveMAC}}
  <p>Active device: {{.ActiveMAC}}</p>
  {{with .Status}}<p>Connected: {{.Connected}} — {{.Name}}</p>{{end}}
  <form method="post" action="/bluetooth/reconnect"><button type="submit">Reconnect</button></form>
{{else}}<p>No active device configured.</p>{{end}}

<h2>Scan for devices</h2>
<form method="post" action="/bluetooth/scan">
  <button type="submit">Scan (~5s)</button>
</form>
{{if .Discovered}}
<table>
  <tr><th>Device</th><th>Name</th><th>Pair</th></tr>
  {{range .Discovered}}
  <tr>
    <td>{{.MAC}}</td><td>{{.Name}}</td>
    <td><form method="post" action="/bluetooth/pair">
      <input type="hidden" name="mac" value="{{.MAC}}">
      <button type="submit">Pair</button>
    </form></td>
  </tr>
  {{end}}
</table>
{{end}}

<h2>Known devices</h2>
<table>
  <tr><th>Device</th><th>Name</th><th>Activate</th></tr>
  {{range .Known}}
  <tr>
    <td>{{.MAC}}</td><td>{{.Name}}</td>
    <td><form method="post" action="/bluetooth/connect">
      <input type="hidden" name="mac" value="{{.MAC}}">
      <button type="submit">Connect</button>
    </form></td>
  </tr>
  {{end}}
</table>
</body>
</html>
```

- [ ] **Step 4: Add the Bluetooth nav link to `cards.html`**

In `system/soundsprout/portal/templates/cards.html`, replace:

```html
<p><a href="/">Home</a></p>
```

with:

```html
<p><a href="/">Home</a> · <a href="/bluetooth">Bluetooth</a></p>
```

- [ ] **Step 5: Update the test helper and add a Bluetooth test**

Replace the `newTestServer` helper in `system/soundsprout/portal/server_test.go` with:

```go
func newTestServer(t *testing.T) (*portal.Server, *portal.MapStore, string) {
	t.Helper()
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	require.NoError(t, os.MkdirAll(filepath.Join(music, "card-1"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(music, "card-1", "a.mp3"), []byte("x"), 0o644))

	mapStore := portal.NewMapStore(
		filepath.Join(dir, "rfid_map.json"),
		filepath.Join(dir, "map.lock"),
	)
	library := portal.NewLibrary(music)
	r := newFakeRunner()
	settings := portal.NewSettings(
		filepath.Join(dir, "conf"),
		filepath.Join(dir, "settings.lock"),
	)
	audio := portal.NewAudioConfig(
		filepath.Join(dir, ".asoundrc"),
		filepath.Join(dir, "cmus-rc"),
		r,
	)
	bt := portal.NewBluetooth(r, settings, audio)
	srv, err := portal.NewServer(mapStore, library, bt, nil)
	require.NoError(t, err)
	return srv, mapStore, music
}
```

Then add this test at the end of `server_test.go`:

```go
func TestBluetoothPageRenders(t *testing.T) {
	srv, _, _ := newTestServer(t)

	req := httptest.NewRequest("GET", "/bluetooth", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Bluetooth")
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./portal/...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add system/soundsprout/portal/server.go system/soundsprout/portal/service.go \
  system/soundsprout/portal/server_test.go system/soundsprout/portal/templates
git commit -m "feat(portal): add bluetooth management UI"
```

---

### Task 7: Deployment (sudoers) and documentation

**Files:**
- Create: `system/sudoers.d/soundsprout-portal`
- Modify: `setup.sh`
- Modify: `README.md`
- Modify: `AGENTS.md`

- [ ] **Step 1: Create the sudoers whitelist**

Create `system/sudoers.d/soundsprout-portal`:

```text
pi ALL=(root) NOPASSWD: /usr/bin/supervisorctl restart btconnect
pi ALL=(root) NOPASSWD: /usr/bin/supervisorctl restart cmus_manager
pi ALL=(root) NOPASSWD: /usr/sbin/alsactl restore
```

- [ ] **Step 2: Install the sudoers file in `setup.sh`**

In `setup.sh`, insert this block immediately before the `supervisorctl reread` line:

```bash
mkdir -p /etc/sudoers.d
cp system/sudoers.d/soundsprout-portal /etc/sudoers.d/soundsprout-portal
chmod 440 /etc/sudoers.d/soundsprout-portal
visudo -cf /etc/sudoers.d/soundsprout-portal
```

- [ ] **Step 3: Verify shell syntax**

Run: `bash -n setup.sh`
Expected: exit 0.

- [ ] **Step 4: Document the Bluetooth tab**

In `README.md`, append this subsection under the `## Web Portal` section:

```markdown
### Bluetooth

The Bluetooth tab (`/bluetooth`) shows the active sink, scans for devices, and can
pair/trust/connect a new speaker or switch the active output. Applying a device rewrites
`/home/pi/.asoundrc` and `/home/pi/.config/cmus/rc` from the `templategen` templates and
restarts `btconnect`/`cmus_manager`/`alsactl` through the narrow passwordless sudo
whitelist in `system/sudoers.d/soundsprout-portal`. The active MAC is stored in the
pi-owned dotenv `/home/pi/.soundsprout/conf`.
```

- [ ] **Step 5: Document the modules in `AGENTS.md`**

In the `## Code structure` list of `AGENTS.md`, add:

```markdown
- `system/soundsprout/bluetooth/`: shared `bluetoothctl` wrapper used by `btconnect` and
  `portal` behind a `Runner` seam.
```

- [ ] **Step 6: Commit**

```bash
git add system/sudoers.d/soundsprout-portal setup.sh README.md AGENTS.md
git commit -m "feat(system): grant scoped restart sudo and document bluetooth"
```

---

### Task 8: Final validation for Part 3

- [ ] **Step 1: Run the full Go suite**

Run: `ASDF_GOLANG_VERSION=1.25.6 make test`
Expected: all packages PASS (`bluetooth`, `btconnect`, `portal`, `templategen`).

- [ ] **Step 2: Run golangci-lint (if available)**

Run: `cd system/soundsprout && golangci-lint run --config .golangci.yml ./...`
Expected: no issues. (If not installed, skip; fix any findings the CI job reports — in
particular a `gosec` G204 that may need a justified `//nolint:gosec` on the exec calls.)

- [ ] **Step 3: Build and vet**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go build ./... && ASDF_GOLANG_VERSION=1.25.6 go vet ./...`
Expected: no output, exit 0.

- [ ] **Step 4: Run Python tests + pre-commit**

Run: `python3 -m pytest -q && pre-commit run --all-files`
Expected: tests PASS; hooks clean.

- [ ] **Step 5: Smoke-test the Bluetooth page**

Run:

```bash
cd system/soundsprout
ASDF_GOLANG_VERSION=1.25.6 go build -o /tmp/soundsprout-server ./cmd
/tmp/soundsprout-server serve --addr 127.0.0.1:8080 &
sleep 1
curl -sS http://127.0.0.1:8080/bluetooth | grep -q "Bluetooth" && echo "OK"
kill %1
```

Expected: prints `OK`.

- [ ] **Step 6: Confirm spec coverage**

Confirm each spec section maps to a task: architecture §4 → Plans 1/3; seams §5 → Plans
1–3; data/API/UI §6 → Plans 2/3; bluetooth/privileges/settings §7 → Tasks 3–7 here; Python
hardening §8 → Part 2 Tasks 4–5; testing §9 → Parts 1–3; error handling §10 → Part 2/3
handlers; security §11 → Part 2 csrfGuard + Task 1 MAC regex; deployment §12 → Plan 1
Task 3 + Task 7 here; docs §13 → Parts 1–3.

- [ ] **Step 7: Commit any final fixups**

```bash
git add -A
git commit -m "chore: part 3 bluetooth validation fixes"
```
