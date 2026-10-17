# Config Single Source + Player Resource Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `/home/pi/.soundsprout/conf` the single source of truth for `PJ_BLUETOOTH_DEVICE`, and stop the Python player from busy-spinning SPI and flooding its log.

**Architecture:** Problem A removes the second config source by pointing every consumer at the pi-owned dotenv (`godotenv.Overload` in the Go CLI; shell scripts source the dotenv; `setup.sh` seeds/validates it and strips stale keys from `/etc/environment`). Problem B moves card reading to a small poll-with-sleep helper in a new hardware-free module, calls cmus status once per LED loop, and demotes per-command logging to DEBUG.

**Tech Stack:** Go 1.21 (`urfave/cli`, `godotenv`, testify), Python 3.9 (pytest, black, flake8), bash, supervisor.

**Spec:** `docs/superpowers/specs/2026-10-17-config-source-and-player-resource-design.md`

## Global Constraints

- **No commits without explicit user authorization.** Repo policy is conservative. Wherever a step says "Checkpoint", stage and show `git status`/`git diff`; do not `git commit` unless the user asks.
- Python formatting: `black` line-length 79, `flake8` max-line-length 80. Run `pre-commit run --all-files` at the end.
- Go tests use `testify`; run `make test` from the repo root.
- The single config location is `/home/pi/.soundsprout/conf` (dotenv). `/etc/environment` must not carry `PJ_BLUETOOTH_DEVICE` or `PJ_DEFAULT_BRANCH`.
- Do not modify the third-party `mfrc522` package or wire the MFRC522 IRQ pin.
- Do not change btconnect's backoff behavior.

---

## File Structure

- `server/rfid_reader.py` (create) — hardware-free `poll_for_card` helper.
- `server/tests/test_rfid_reader.py` (create) — fake reader + recording sleep.
- `server/player.py` (modify) — use `poll_for_card`; drop warnings-as-errors; demote a log.
- `server/cmus_utils.py` (modify) — per-command logs INFO→DEBUG.
- `server/peripheral_helpers.py` (modify) — one `cmus_status()` per LED loop; drop `music_is_*` wrappers.
- `system/scripts/seed_config.sh` (create) — seed/validate the dotenv; strip stale keys.
- `server/tests/test_seed_config.py` (create) — subprocess test of the seed script.
- `setup.sh` (modify) — call the seed script.
- `system/scripts/gitpull.sh` (modify) — source the dotenv instead of `/etc/environment`.
- `system/scripts/cmus.sh` (modify) — drop the unused `/etc/environment` source.
- `system/soundsprout/cmd/main.go` (modify) — `godotenv.Load` → `godotenv.Overload`.
- `system/soundsprout/portal/bluetooth_test.go` (modify) — assert all consumer files agree.
- `.env.sample`, `README.md`, `docs/rfid-map-and-bluetooth.md` (modify) — single-location docs.

---

## Task 1: Non-blocking RFID polling helper

**Files:**
- Create: `server/rfid_reader.py`
- Create: `server/tests/test_rfid_reader.py`
- Modify: `server/player.py:1-30,120-124`

**Interfaces:**
- Produces: `rfid_reader.poll_for_card(reader, poll_interval=0.05, sleep=time.sleep) -> (id, text)`. `reader` must expose `read_no_block()` returning `(id, text)`, with `id` falsy when no card is present.

- [ ] **Step 1: Write the failing test**

Create `server/tests/test_rfid_reader.py`:

```python
from rfid_reader import poll_for_card


class FakeReader:
    def __init__(self, results):
        self._results = list(results)
        self.calls = 0

    def read_no_block(self):
        self.calls += 1
        return self._results.pop(0)


def test_returns_first_detected_card():
    reader = FakeReader([(None, None), (None, None), (42, "hello")])
    sleeps = []

    result = poll_for_card(reader, poll_interval=0.05, sleep=sleeps.append)

    assert result == (42, "hello")
    assert reader.calls == 3


def test_sleeps_between_empty_polls():
    reader = FakeReader([(None, None), (None, None), (7, "x")])
    sleeps = []

    poll_for_card(reader, poll_interval=0.05, sleep=sleeps.append)

    assert sleeps == [0.05, 0.05]


def test_does_not_sleep_after_detecting():
    reader = FakeReader([(1, "a")])
    sleeps = []

    assert poll_for_card(reader, poll_interval=0.05, sleep=sleeps.append) == (
        1,
        "a",
    )
    assert sleeps == []
```

- [ ] **Step 2: Run test to verify it fails**

Run: `python3 -m pytest server/tests/test_rfid_reader.py -v`
Expected: FAIL — `ModuleNotFoundError: No module named 'rfid_reader'`.

- [ ] **Step 3: Write minimal implementation**

Create `server/rfid_reader.py`:

```python
import time

DEFAULT_POLL_INTERVAL = 0.05


def poll_for_card(reader, poll_interval=DEFAULT_POLL_INTERVAL, sleep=time.sleep):
    """Return (id, text) once a card is present, sleeping between polls.

    ``reader`` must expose ``read_no_block()``; a falsy id means no card.
    Sleeping between polls keeps the process off the CPU while idle, unlike
    ``SimpleMFRC522.read()`` which spins on SPI with no pause.
    """
    while True:
        rfid_id, text = reader.read_no_block()
        if rfid_id:
            return rfid_id, text
        sleep(poll_interval)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `python3 -m pytest server/tests/test_rfid_reader.py -v`
Expected: PASS (3 passed).

- [ ] **Step 5: Wire the player to the helper and drop warnings-as-errors**

In `server/player.py`:

- Delete `import warnings` (line 11) and `warnings.simplefilter("error")` (line 26).
- Add `from rfid_reader import poll_for_card` to the imports.
- Replace the read call. Change:

```python
            # Check for RFID card
            rfid_id, text = rfid_reader.read()
```

to:

```python
            # Check for RFID card
            rfid_id, text = poll_for_card(rfid_reader)
```

- Demote the per-iteration map log. Change line 114:

```python
        Logger.info(f"Loading map file {RFID_TO_MUSIC_MAP}")
```

to:

```python
        Logger.debug(f"Loading map file {RFID_TO_MUSIC_MAP}")
```

- [ ] **Step 6: Lint and run the Python suite**

Run: `black server/rfid_reader.py server/player.py server/tests/test_rfid_reader.py && flake8 server/ && python3 -m pytest`
Expected: black reformats if needed, flake8 clean, all pytest tests pass.

- [ ] **Step 7: Checkpoint**

Run: `git status && git diff server/rfid_reader.py server/player.py server/tests/test_rfid_reader.py`
Expected: the new module/test and the three `player.py` edits. Do not commit.

---

## Task 2: Quiet cmus status + LED loop

**Files:**
- Modify: `server/cmus_utils.py:36,60`
- Modify: `server/peripheral_helpers.py:123-132,167-197`

**Interfaces:**
- Consumes: `cmus_status()` already returns `(is_playing, is_shuffle, is_repeat)` (`server/cmus_utils.py:84-91`).
- Produces: no new public interface; `music_is_playing`/`music_is_shuffling`/`music_is_repeating` are removed.

- [ ] **Step 1: Demote the per-command logs**

In `server/cmus_utils.py`, change the always-on INFO lines inside `send_to_cmus_socket`:

Line 36:

```python
                    Logger.info(f"Using CMUS socket path: {cmus_socket_path}")
```

to:

```python
                    Logger.debug(f"Using CMUS socket path: {cmus_socket_path}")
```

Line 60:

```python
                Logger.info(f"Completed command: {cmd}")
```

to:

```python
                Logger.debug(f"Completed command: {cmd}")
```

Leave the guarded `Sending command` INFO (lines 31-32) and every `Logger.critical`/`error` path unchanged.

- [ ] **Step 2: Call cmus_status once per LED loop and drop the wrappers**

In `server/peripheral_helpers.py`, delete the three wrappers (lines 123-132):

```python
def music_is_playing():
    return cmus_status()[0]


def music_is_shuffling():
    return cmus_status()[1]


def music_is_repeating():
    return cmus_status()[2]
```

Replace the body of `led_update_loop` (the `try` block, lines 171-192) so it fetches status once:

```python
            try:
                is_playing, is_shuffle, is_repeat = cmus_status()

                if is_playing:
                    GPIO.output(PLAY_LED_PIN, GPIO.HIGH)
                    time.sleep(0.5)  # LED is on for 0.5 seconds
                    GPIO.output(PLAY_LED_PIN, GPIO.LOW)
                    time.sleep(0.5)  # LED is off for 0.5 seconds
                elif ensure_is_cmus_running():
                    GPIO.output(PLAY_LED_PIN, GPIO.HIGH)
                else:
                    GPIO.output(PLAY_LED_PIN, GPIO.LOW)

                if is_shuffle:
                    GPIO.output(SHUFFLE_LED_PIN, GPIO.HIGH)
                else:
                    GPIO.output(SHUFFLE_LED_PIN, GPIO.LOW)

                if is_repeat:
                    GPIO.output(REPEAT_LED_PIN, GPIO.HIGH)
                else:
                    GPIO.output(REPEAT_LED_PIN, GPIO.LOW)

                time.sleep(0.5)  # you can adjust the sleep time as needed
            except Exception as e:
                Logger.error(f"Error in LED Loop: {e}")
                raise e
```

`cmus_status` stays imported; `ensure_is_cmus_running` stays imported and used.

- [ ] **Step 3: Verify no stale references and lint**

Run: `grep -rn "music_is_" server/ || echo "none"` and `black server/peripheral_helpers.py server/cmus_utils.py && flake8 server/`
Expected: grep prints `none`; black reformats if needed; flake8 clean.

- [ ] **Step 4: Run the suite to confirm nothing broke**

Run: `python3 -m pytest`
Expected: all tests pass (these modules are linted, not imported, by the suite).

- [ ] **Step 5: Checkpoint**

Run: `git diff server/cmus_utils.py server/peripheral_helpers.py`
Expected: two INFO→DEBUG changes and the single-status LED loop with wrappers removed. Do not commit.

---

## Task 3: Portal consistency test + dotenv precedence

**Files:**
- Modify: `system/soundsprout/portal/bluetooth_test.go:28-38`
- Modify: `system/soundsprout/cmd/main.go:26`

**Interfaces:**
- Consumes: `portal.NewBluetooth(runner, settings, audio)`, `settings.Get("PJ_BLUETOOTH_DEVICE")`, and the rendered files at the `asoundrc`/`cmus-rc` paths passed to `portal.NewAudioConfig`.

- [ ] **Step 1: Extend the portal test to assert every consumer file agrees**

In `system/soundsprout/portal/bluetooth_test.go`, replace `newTestBluetooth` and `TestBluetoothActivatePersistsAndApplies` so the test can also read the rendered files (add `os` to the import block):

```go
import (
	"os"
	"path/filepath"
	"testing"

	"aaronromeo.com/soundsprout/portal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestBluetooth(t *testing.T) (*portal.Bluetooth, *fakeRunner, *portal.Settings, string, string) {
	t.Helper()
	dir := t.TempDir()
	r := newFakeRunner()
	asoundrc := filepath.Join(dir, ".asoundrc")
	cmusrc := filepath.Join(dir, "cmus-rc")
	settings := portal.NewSettings(
		filepath.Join(dir, "conf"),
		filepath.Join(dir, "settings.lock"),
	)
	audio := portal.NewAudioConfig(asoundrc, cmusrc, r)
	return portal.NewBluetooth(r, settings, audio), r, settings, asoundrc, cmusrc
}

func TestBluetoothActivatePersistsAndApplies(t *testing.T) {
	bt, r, settings, asoundrc, cmusrc := newTestBluetooth(t)
	require.NoError(t, bt.Activate("AA:BB:CC:DD:EE:FF"))

	mac, err := settings.Get("PJ_BLUETOOTH_DEVICE")
	require.NoError(t, err)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", mac)

	asound, err := os.ReadFile(asoundrc)
	require.NoError(t, err)
	assert.Contains(t, string(asound), `device "AA:BB:CC:DD:EE:FF"`)

	cmus, err := os.ReadFile(cmusrc)
	require.NoError(t, err)
	assert.Contains(t, string(cmus), "bluealsa:DEV=AA:BB:CC:DD:EE:FF")

	assert.Contains(t, r.runs, "bluetoothctl connect AA:BB:CC:DD:EE:FF")
	assert.Contains(t, r.runs, "sudo supervisorctl restart btconnect")
}

func TestBluetoothActivateRejectsBadMAC(t *testing.T) {
	bt, _, _, _, _ := newTestBluetooth(t)
	require.Error(t, bt.Activate("nope"))
}
```

- [ ] **Step 2: Run the test to verify it passes against current production code**

Run: `cd system/soundsprout && go test ./portal/... -run TestBluetooth -v`
Expected: PASS. `persistAndApply` already writes all three files; this locks the invariant.

- [ ] **Step 3: Make the dotenv authoritative in the process environment**

In `system/soundsprout/cmd/main.go`, change line 26:

```go
	err := godotenv.Load("/home/pi/.soundsprout/conf")
```

to:

```go
	err := godotenv.Overload("/home/pi/.soundsprout/conf")
```

This makes the dotenv win over any stale value inherited from the process environment (e.g. PAM-loaded `/etc/environment` on an interactive shell).

- [ ] **Step 4: Run the Go tests**

Run: `cd system/soundsprout && go build ./... && go test ./...` then from repo root `make test`
Expected: build ok; all tests pass.

- [ ] **Step 5: Checkpoint**

Run: `git diff system/soundsprout/portal/bluetooth_test.go system/soundsprout/cmd/main.go`
Expected: the consistency test and the one-word `Load`→`Overload` change. Do not commit.

---

## Task 4: Seed/validate the dotenv; point shell scripts at it

**Files:**
- Create: `system/scripts/seed_config.sh`
- Create: `server/tests/test_seed_config.py`
- Modify: `setup.sh:16-29`
- Modify: `system/scripts/gitpull.sh:8`
- Modify: `system/scripts/cmus.sh:8`

**Interfaces:**
- Produces: `system/scripts/seed_config.sh`, honoring env overrides `SOUNDSPROUT_CONF_DIR` (default `/home/pi/.soundsprout`) and `SYSTEM_ENV_FILE` (default `/etc/environment`); always exits 0.

- [ ] **Step 1: Write the failing test**

Create `server/tests/test_seed_config.py`:

```python
import os
import subprocess
from pathlib import Path

SCRIPT = (
    Path(__file__).resolve().parents[2]
    / "system"
    / "scripts"
    / "seed_config.sh"
)


def run(conf_dir, env_file):
    env = dict(
        os.environ,
        SOUNDSPROUT_CONF_DIR=str(conf_dir),
        SYSTEM_ENV_FILE=str(env_file),
    )
    return subprocess.run(
        ["bash", str(SCRIPT)], env=env, capture_output=True, text=True
    )


def test_creates_conf_and_warns_when_unset(tmp_path):
    env_file = tmp_path / "environment"
    env_file.write_text("PATH=/usr/bin\n")
    conf_dir = tmp_path / ".soundsprout"

    result = run(conf_dir, env_file)

    assert result.returncode == 0
    assert (conf_dir / "conf").exists()
    assert "PJ_BLUETOOTH_DEVICE is not set" in result.stderr


def test_placeholder_mac_warns(tmp_path):
    env_file = tmp_path / "environment"
    env_file.write_text("PATH=/usr/bin\n")
    conf_dir = tmp_path / ".soundsprout"
    conf_dir.mkdir()
    (conf_dir / "conf").write_text(
        "PJ_BLUETOOTH_DEVICE=XX:XX:XX:XX:XX:XX\n"
    )

    result = run(conf_dir, env_file)

    assert result.returncode == 0
    assert "PJ_BLUETOOTH_DEVICE is not set" in result.stderr


def test_configured_mac_does_not_warn(tmp_path):
    env_file = tmp_path / "environment"
    env_file.write_text("PATH=/usr/bin\n")
    conf_dir = tmp_path / ".soundsprout"
    conf_dir.mkdir()
    (conf_dir / "conf").write_text("PJ_BLUETOOTH_DEVICE=AA:BB:CC:DD:EE:FF\n")

    result = run(conf_dir, env_file)

    assert result.returncode == 0
    assert "WARNING" not in result.stderr


def test_strips_project_keys_from_system_env(tmp_path):
    env_file = tmp_path / "environment"
    env_file.write_text(
        "PATH=/usr/bin\n"
        "PJ_BLUETOOTH_DEVICE=AA:BB:CC:DD:EE:FF\n"
        "PJ_DEFAULT_BRANCH=main\n"
        "OTHER=1\n"
    )
    conf_dir = tmp_path / ".soundsprout"
    conf_dir.mkdir()
    (conf_dir / "conf").write_text("PJ_BLUETOOTH_DEVICE=AA:BB:CC:DD:EE:FF\n")

    result = run(conf_dir, env_file)

    assert result.returncode == 0
    text = env_file.read_text()
    assert "PJ_BLUETOOTH_DEVICE" not in text
    assert "PJ_DEFAULT_BRANCH" not in text
    assert "OTHER=1" in text
    assert (tmp_path / "environment.bak").exists()
```

- [ ] **Step 2: Run test to verify it fails**

Run: `python3 -m pytest server/tests/test_seed_config.py -v`
Expected: FAIL — `seed_config.sh` does not exist (bash exits non-zero).

- [ ] **Step 3: Write the seed script**

Create `system/scripts/seed_config.sh`:

```bash
#!/bin/bash
# Seed and validate the SoundSprout dotenv, the single source of truth for
# PJ_BLUETOOTH_DEVICE and PJ_DEFAULT_BRANCH. Invoked by setup.sh.
#
# Deliberately does NOT use `set -e`: it must never fail a caller's update,
# and a missing MAC is a warning (the operator pairs via the portal later).
set -uo pipefail

CONF_DIR="${SOUNDSPROUT_CONF_DIR:-/home/pi/.soundsprout}"
CONF_FILE="$CONF_DIR/conf"
ENV_FILE="${SYSTEM_ENV_FILE:-/etc/environment}"
OWNER="${SOUNDSPROUT_OWNER:-pi}"

mkdir -p "$CONF_DIR"
if [ ! -f "$CONF_FILE" ]; then
    : > "$CONF_FILE"
fi

if [ "$(id -u)" -eq 0 ]; then
    chown -R "$OWNER:" "$CONF_DIR" 2>/dev/null || true
fi
chmod 600 "$CONF_FILE" 2>/dev/null || true

# Remove project keys from /etc/environment so nothing shadows the dotenv.
if [ -f "$ENV_FILE" ] && grep -qE '^(PJ_BLUETOOTH_DEVICE|PJ_DEFAULT_BRANCH)=' "$ENV_FILE"; then
    cp "$ENV_FILE" "$ENV_FILE.bak"
    grep -vE '^(PJ_BLUETOOTH_DEVICE|PJ_DEFAULT_BRANCH)=' "$ENV_FILE" > "$ENV_FILE.tmp" || true
    mv "$ENV_FILE.tmp" "$ENV_FILE"
    echo "Removed PJ_BLUETOOTH_DEVICE/PJ_DEFAULT_BRANCH from $ENV_FILE (backup: $ENV_FILE.bak)."
fi

# Validate.
mac="$(grep -E '^PJ_BLUETOOTH_DEVICE=' "$CONF_FILE" | tail -n1 | cut -d= -f2- || true)"
if [ -z "$mac" ] || [ "$mac" = "XX:XX:XX:XX:XX:XX" ]; then
    echo "WARNING: PJ_BLUETOOTH_DEVICE is not set in $CONF_FILE." >&2
    echo "WARNING: Open the portal (http://<pi>:8080/bluetooth) to pair a speaker," >&2
    echo "WARNING: or write PJ_BLUETOOTH_DEVICE=<MAC> to $CONF_FILE." >&2
fi

exit 0
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `python3 -m pytest server/tests/test_seed_config.py -v`
Expected: PASS (4 passed).

- [ ] **Step 5: Call the seed script from setup.sh**

In `setup.sh`, after the `supervisorctl reread`/`update` lines (end of file), add:

```bash
bash system/scripts/seed_config.sh
```

- [ ] **Step 6: Point gitpull.sh at the dotenv**

In `system/scripts/gitpull.sh`, replace line 8 (`source /etc/environment`) with:

```bash
if [ -f /home/pi/.soundsprout/conf ]; then
    set -a
    source /home/pi/.soundsprout/conf
    set +a
fi
```

- [ ] **Step 7: Drop the unused source from cmus.sh**

In `system/scripts/cmus.sh`, delete line 8 (`source /etc/environment`).

- [ ] **Step 8: Verify the scripts and seed logic**

Run: `bash -n system/scripts/seed_config.sh setup.sh system/scripts/gitpull.sh system/scripts/cmus.sh && python3 -m pytest`
Expected: `bash -n` prints nothing (syntax ok); all pytest tests pass.

- [ ] **Step 9: Checkpoint**

Run: `git status && git diff system/scripts/ setup.sh`
Expected: new `seed_config.sh`, the `setup.sh` call, gitpull's guarded source, cmus.sh source removed. Do not commit.

---

## Task 5: Documentation for the single config source

**Files:**
- Modify: `.env.sample:1-3`
- Modify: `README.md:226,318,322-329`
- Modify: `docs/rfid-map-and-bluetooth.md:243-246`

- [ ] **Step 1: Update `.env.sample`**

Replace lines 1-2:

```
# Copy to /etc/environment and/or /home/pi/.soundsprout/conf (dotenv loaded by soundsprout-server).
# See README step 16.
```

with:

```
# This file is the single source of truth for the values below. The portal
# writes them to /home/pi/.soundsprout/conf, which soundsprout-server loads and
# system/scripts/*.sh source. Do not also add these to /etc/environment.
# See README step 16.
```

- [ ] **Step 2: Update README step 16**

Replace the step 16 body (lines 324-329) with a single-location description:

```
See `.env.sample` for the expected keys. `PJ_BLUETOOTH_DEVICE` is required by the Go `btconnect`/`templategen` commands; `PJ_DEFAULT_BRANCH` is used by `system/scripts/gitpull.sh` (defaults to `main`). Both live in the single dotenv `/home/pi/.soundsprout/conf`, which `soundsprout-server` loads and `system/scripts/gitpull.sh` sources. Do **not** also set them in `/etc/environment` — `setup.sh` strips them from there if found.

The normal way to set the speaker is the web portal (`/bluetooth`), which writes the dotenv and the audio configs. To set it by hand instead:

```bash
mkdir -p /home/pi/.soundsprout
printf "PJ_BLUETOOTH_DEVICE=XX:XX:XX:XX:XX:XX\n" > /home/pi/.soundsprout/conf
```

`setup.sh` creates the dotenv if missing and warns when `PJ_BLUETOOTH_DEVICE` is unset.
```

- [ ] **Step 3: Update the README step 8 preamble**

Line 226 says "Set `PJ_BLUETOOTH_DEVICE` to the MAC address of your Bluetooth audio device (see step 16)." Append: " It lives in the dotenv `/home/pi/.soundsprout/conf` (see step 16)."

- [ ] **Step 4: Update the config-location table**

In `docs/rfid-map-and-bluetooth.md`, replace the `/etc/environment` row (line 246):

```
| `/etc/environment` | System-wide env vars including `PJ_BLUETOOTH_DEVICE` |
```

with:

```
| `/etc/environment` | System env file; **not** used for `PJ_BLUETOOTH_DEVICE`/`PJ_DEFAULT_BRANCH` (they live only in the dotenv above) |
```

- [ ] **Step 5: Verify docs format**

Run: `pre-commit run --files .env.sample README.md docs/rfid-map-and-bluetooth.md docs/superpowers/specs/2026-10-17-config-source-and-player-resource-design.md docs/superpowers/plans/2026-10-17-config-source-and-player-resource.md`
Expected: trailing-whitespace and end-of-file-fixer pass (or are auto-fixed; re-run if needed).

- [ ] **Step 6: Checkpoint**

Run: `git diff .env.sample README.md docs/rfid-map-and-bluetooth.md`
Expected: the doc changes above. Do not commit.

---

## Task 6: Full quality gate

- [ ] **Step 1: Run every gate**

```bash
python3 -m pytest
make test
(cd system/soundsprout && golangci-lint run --config .golangci.yml ./...)
pre-commit run --all-files
```

Expected: pytest passes; `make test` passes and writes coverage; golangci-lint clean; pre-commit passes (fix and re-run if it auto-edits).

- [ ] **Step 2: Confirm the single-source invariant by hand**

Run: `grep -rn "etc/environment" setup.sh system/scripts/ README.md .env.sample docs/rfid-map-and-bluetooth.md`
Expected: only the `seed_config.sh` stripping logic and the docs that say it is *not* used. No `source /etc/environment` remains in `system/scripts/`.

- [ ] **Step 3: Report**

Summarize changed files, gate results, and the on-hardware checks deferred to bead `pyrfid-jukebox-nfj`. Do not commit or push.

---

## Self-Review

- **Spec coverage:** Problem A → Tasks 3 (Overload, portal invariant), 4 (seed script, gitpull, cmus.sh), 5 (docs); Problem B → Tasks 1 (reader), 2 (logs/LED). Verification task 6. ✓
- **Placeholders:** none — every code and command step is concrete. ✓
- **Type consistency:** `poll_for_card(reader, poll_interval, sleep)` used identically in tasks 1; `cmus_status()` tuple order `(is_playing, is_shuffle, is_repeat)` matches `cmus_utils.py:88-91`; `seed_config.sh` env overrides (`SOUNDSPROUT_CONF_DIR`, `SYSTEM_ENV_FILE`) match the test. ✓
