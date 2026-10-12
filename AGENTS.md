# AGENTS.md

## Overview
This is a Raspberry Pi RFID jukebox (SoundSprout). The repo contains:
- **Go service (`system/soundsprout/`)**: CLI (`soundsprout-server`) with `btconnect` (Bluetooth connection maintenance) and `templategen` (template generation for cmus/asoundrc).
- **Python server (`server/`)**: RFID-based player using MFRC522, RPi.GPIO, cmus control.
- **System configs**: Supervisor configs (`system/supervisor/conf.d/`), scripts (`system/scripts/`), sounds.

## Quick start
- Go module: `system/soundsprout/go.mod` (Go 1.21.1; CI uses ~1.21; `.tool-versions` has 1.23.1)
- Python deps: `requirements.txt` (Flask, mfrc522, gpiozero, lgpio, deezer-py, etc.)
- Setup script: `sudo ./setup.sh` (copies scripts, installs supervisor configs, sets up logs)

## Build, test, lint, format
**Top-level Makefile** (delegates to `system/soundsprout/`):
```bash
make build  # builds soundsprout binary (go build ./cmd -> soundsprout-server)
make test   # runs go tests with coverage (coverprofile in repo root, cover.html generated)
make clean  # clean in soundsprout
make install # installs to /usr/local/bin (GOBIN=/usr/local/bin)
make run    # runs local binary
```

**Go (soundsprout):**
- Module: `aaronromeo.com/soundsprout`
- Tests: `make -C system/soundsprout test` or root `make test`
- Lint: `golangci-lint run --config system/soundsprout/.golangci.yml` (version v1.54 in CI)
- Working dir for Go linting in CI is `./system/soundsprout`

**Python:**
- Pre-commit hooks configured (`.pre-commit-config.yaml`): black (23.11.0), flake8 (6.1.0), trailing whitespace, end-of-file-fixer, check-yaml, check-added-large-files
- Run: `pre-commit run --all-files` (also runs in CI on push)

## Code structure
- `system/soundsprout/cmd/main.go`: CLI with commands `btconnect` (maintains BT connection, loops every 15s, needs `PJ_BLUETOOTH_DEVICE`) and `templategen` (generates cmus/asoundrc templates from embedded templates; writes runner script to output dir).
- `system/soundsprout/btconnect/service.go`: checks `bluetoothctl info <device>` for "Connected: yes", reconnects via `bluetoothctl connect <device>` if needed. Uses `exec.Command` (interface for testability).
- `system/soundsprout/templategen/`: template generation. Templates in `templates/*.txt` use Go templates with `{{.PJ_BLUETOOTH_DEVICE}}`. Generates runner.sh with commands to create dirs, move generated files, chown pi, restart supervisor services (cmus_manager, btconnect), and run `alsactl restore` as needed.
- `system/soundsprout/portal/`: Go web portal package (embedded HTML templates). Started
  with `soundsprout-server serve`.
- `server/player.py`: RFID jukebox using `mfrc522.SimpleMFRC522`, `RPi.GPIO`, cmus via `cmus_utils`. Acquires lock at `/tmp/pyrfid_jukebox.lock`, loads `server/rfid_map.json`, plays folder contents via cmus when RFID matches a mapped folder. Handles buttons/LEDs via `peripheral_helpers`.
- `server/cmus_utils.py`, `server/logger.py`, `server/peripheral_helpers.py`: supporting modules.
- Supervisor configs: `soundsprout_server.conf` runs Python `server/player.py` (autostart=false in config), `btconnect.conf`, `cmus_manager.conf`, `gitpull.conf`, `prepare_logs.conf`. Setup copies them to `/etc/supervisor/conf.d/`.

## Environment & conventions
- Env vars: `PJ_BLUETOOTH_DEVICE` (required by btconnect and templategen). `.env.sample` exists.
- Python formatting: black (line-length 79), flake8 (max-line-length 80).
- Go: golangci-lint configured (`system/soundsprout/.golangci.yml`), strict but not extreme. Tests use testify.
- Project uses pre-commit; ensure hooks pass before committing.

## CI
- `.github/workflows/testing.yml`: runs `make test`, uploads coverage and test output, posts Go test output as PR comment (on PRs).
- `.github/workflows/linting.yml`: runs golangci-lint (v1.54) with config in `./system/soundsprout/.golangci.yml`, and runs `pre-commit run --all-files`.

## Common workflows
- Run all Go tests from root: `make test` (generates `cover.out` and `cover.html` in repo root)
- Run a single Go test: `cd system/soundsprout && go test -v ./templategen/... -run TestRun` (example)
- Run golangci-lint locally: `golangci-lint run --config system/soundsprout/.golangci.yml ./...` from `system/soundsprout/`
- Run pre-commit: `pre-commit run --all-files`
- Template generation: `soundsprout-server templategen --output-dir <dir>` (requires `PJ_BLUETOOTH_DEVICE` env var; templates use it)
- Bluetooth maintenance: `soundsprout-server btconnect` (loops, requires `PJ_BLUETOOTH_DEVICE`)

## Notes
- Go service is the primary buildable component tested in CI (Python has no explicit test suite configured in CI).
- Supervisor-managed services; `setup.sh` is intended to run as root on the Pi.
- RFID map lives at `server/rfid_map.json` (created if missing). The player writes mappings when unknown RFID is seen.
- Lock file: `/tmp/pyrfid_jukebox.lock` prevents multiple player instances.
- Generated artifacts: `cover.out`, `cover.html`, `soundsprout-server` binary in `system/soundsprout/`.
