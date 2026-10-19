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
make install # builds soundsprout-server to /usr/local/bin (needs sudo; pass GOCMD=<go path> if go is not on root's PATH)
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
- Tests: `python3 -m pytest` (config in `pyproject.toml`, tests under `server/tests/`)

## Code structure
- `system/soundsprout/cmd/main.go`: CLI with commands `btconnect` (maintains BT connection, 15s loop with exponential backoff up to 300s on failures, needs `PJ_BLUETOOTH_DEVICE`) and `templategen` (generates cmus/asoundrc templates from embedded templates; writes runner script to output dir).
- `system/soundsprout/btconnect/service.go`: checks `bluetoothctl info <device>` for "Connected: yes", reconnects via `bluetoothctl connect <device>` if needed. Uses `exec.Command` (interface for testability).
- `system/soundsprout/templategen/`: template generation. Templates in `templates/*.txt` use Go templates with `{{.PJ_BLUETOOTH_DEVICE}}`. Generates runner.sh with commands to create dirs, move generated files, chown pi, restart supervisor services (cmus_manager, btconnect), and run `alsactl restore` as needed.
- `system/soundsprout/portal/`: Go web portal package (embedded HTML templates). Started
  with `soundsprout-server serve`.
- `system/soundsprout/bluetooth/`: shared `bluetoothctl` wrapper used by `btconnect` and
  `portal` behind a `Runner` seam.
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

<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:46cd31e7 -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

**Architecture in one line:** issues live in a local Dolt DB; sync uses `refs/dolt/data` on your git remote; `.beads/issues.jsonl` is a passive export. See https://github.com/gastownhall/beads/blob/main/docs/core-concepts/sync-concepts.md for details and anti-patterns.

## Agent Context Profiles

The managed Beads block is task-tracking guidance, not permission to override repository, user, or orchestrator instructions.

- **Conservative (default)**: Use `bd` for task tracking. Do not run git commits, git pushes, or Dolt remote sync unless explicitly asked. At handoff, report changed files, validation, and suggested next commands.
- **Minimal**: Keep tool instruction files as pointers to `bd prime`; use the same conservative git policy unless active instructions say otherwise.
- **Team-maintainer**: Only when the repository explicitly opts in, agents may close beads, run quality gates, commit, and push as part of session close. A current "do not commit" or "do not push" instruction still wins.

## Session Completion

This protocol applies when ending a Beads implementation workflow. It is subordinate to explicit user, repository, and orchestrator instructions.

1. **File issues for remaining work** - Create beads for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **Handle git/sync by active profile**:
   ```bash
   # Conservative/minimal/default: report status and proposed commands; wait for approval.
   git status

   # Team-maintainer opt-in only, unless current instructions forbid it:
   git pull --rebase
   bd dolt push
   git push
   git status
   ```
5. **Hand off** - Summarize changes, validation, issue status, and any blocked sync/commit/push step

**Critical rules:**
- Explicit user or orchestrator instructions override this Beads block.
- Do not commit or push without clear authority from the active profile or the current user request.
- If a required sync or push is blocked, stop and report the exact command and error.
<!-- END BEADS INTEGRATION -->

<!-- BEGIN BEADS CODEX SETUP: generated by bd setup codex -->
## Beads Issue Tracker

Use Beads (`bd`) for durable task tracking in repositories that include it. Use the `beads` skill at `.agents/skills/beads/SKILL.md` (project install) or `~/.agents/skills/beads/SKILL.md` (global install) for Beads workflow guidance, then use the `bd` CLI for issue operations.

### Quick Reference

```bash
bd ready                # Find available work
bd show <id>            # View issue details
bd update <id> --claim  # Claim work
bd close <id>           # Complete work
bd prime                # Refresh Beads context
```

### Rules

- Use `bd` for all task tracking; do not create markdown TODO lists.
- Run `bd prime` when Beads context is missing or stale. Codex 0.129.0+ can load Beads context automatically through native hooks; use `/hooks` to inspect or toggle them.
- Keep persistent project memory in Beads via `bd remember`; do not create ad hoc memory files.

**Architecture in one line:** issues live in a local Dolt DB; sync uses `refs/dolt/data` on your git remote; `.beads/issues.jsonl` is a passive export. See https://github.com/gastownhall/beads/blob/main/docs/core-concepts/sync-concepts.md for details and anti-patterns.
<!-- END BEADS CODEX SETUP -->
