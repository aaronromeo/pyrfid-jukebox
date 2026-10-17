# Config Single Source + Player Resource Fixes — Design

**Date:** 2026-10-17
**Status:** Approved
**Background:** Handoff at
`/tmp/opencode/pyrfid-jukebox-config-cpu-handoff.md` (2026-10-16). Related
incident: the sibling box `madman` went silent after a speaker swap because
`PJ_BLUETOOTH_DEVICE` was overwritten in only one of two sources.

## Problem

Two independent defects in the SoundSprout repo.

**A. `PJ_BLUETOOTH_DEVICE` has two sources of truth and `setup.sh` seeds
neither.**

| Location | Written by | Read by |
|---|---|---|
| `/home/pi/.soundsprout/conf` (dotenv) | portal `Settings.Set` | Go `soundsprout-server` (`godotenv.Load`, `cmd/main.go`), portal `Settings.Get` |
| `/etc/environment` | nobody in the repo (manual, per README step 16) | `system/scripts/gitpull.sh`, `system/scripts/cmus.sh` (`source`) |

Facts confirmed by reading the code:

- Supervisor configs do **not** set or load `PJ_BLUETOOTH_DEVICE`; supervisor
  does not read `/etc/environment`. The Go service therefore already gets the
  value only from the dotenv (`godotenv.Load` in `cmd/main.go:26`).
- `cmus.sh` sources `/etc/environment` but references none of its variables
  (`XDG_RUNTIME_DIR` is set locally).
- `gitpull.sh` sources `/etc/environment` only for `PJ_DEFAULT_BRANCH`.
- `godotenv.Load` does **not** override variables already present in the
  process environment. `/etc/environment` is loaded by PAM for interactive
  logins, so a stale value there can shadow the dotenv for manual runs — the
  exact class of bug that broke `madman`.
- The portal's `persistAndApply` (`portal/bluetooth.go:77`) writes the dotenv
  and the audio configs but never `/etc/environment`; a speaker change outside
  the portal updates nothing the repo manages.

**B. The Python player burns ~51% CPU and floods/rotates its log every ~6
minutes.**

- `player.py:122` calls `SimpleMFRC522.read()`, which busy-spins
  `read_no_block()` (SPI poll, no sleep) — pegs one core.
- `peripheral_helpers.led_update_loop_factory` calls `music_is_playing()`,
  `music_is_shuffling()`, `music_is_repeating()` each ~0.5 s. Each calls
  `cmus_status()`, which opens a new cmus socket connection — 3 round-trips per
  iteration for the same status.
- `cmus_utils.send_to_cmus_socket` logs `Using CMUS socket path` and
  `Completed command` at INFO on every call, including `status` — ~6 INFO
  lines/s, which floods the log (`soundsprout_server.conf` rotates at 200 KB).
- `player.py:26` `warnings.simplefilter("error")` promotes warnings to
  exceptions; an `RPi.GPIO.setup()` warning became a fatal crash-loop on the Pi.
- `player.py:114` logs `Loading map file ...` every loop iteration.

## Goal

1. One source of truth for `PJ_BLUETOOTH_DEVICE` (and `PJ_DEFAULT_BRANCH`)
   that `setup.sh` seeds/validates, so a fresh install cannot come up
   misconfigured and a speaker change is a single edit.
2. The player idles near-zero CPU and stops flooding its log, without hurting
   tap latency or removing useful error signal.

## Decisions (confirmed with user)

1. **The dotenv `/home/pi/.soundsprout/conf` is authoritative.** The portal
   already owns it; `godotenv` already loads it for the Go service. No sudo
   needed for portal writes, fewest moving parts.
2. **`setup.sh` seeds the dotenv and warns (does not fail) when the MAC is
   unset.** A fresh install has no MAC until the operator pairs via the portal,
   and `gitpull.sh` runs `bash setup.sh` on every update, so a non-zero exit
   would abort updates.
3. **Reader polling uses `read_no_block()` + a short `time.sleep`.** No
   wiring, no register-level code, testable with a fake reader.
4. **Remove `warnings.simplefilter("error")`.** Warning-driven crash-loops in a
   long-running hardware service are not worth it.
5. Work continues on the current branch (`feat/btconnect-resilience`).

## Design

### Problem A — dotenv single source

**Data flow after the change:** the portal writes the MAC to the dotenv only;
`soundsprout-server` (btconnect/templategen) and `gitpull.sh` read the dotenv;
`/etc/environment` no longer carries `PJ_BLUETOOTH_DEVICE` or
`PJ_DEFAULT_BRANCH`.

1. **`system/soundsprout/cmd/main.go`** — change `godotenv.Load(...)` to
   `godotenv.Overload(...)` so the dotenv wins even when a stale value is
   present in the process environment (e.g. PAM-loaded `/etc/environment` on an
   interactive SSH shell). One-line change.

2. **`system/scripts/gitpull.sh`** — replace `source /etc/environment` with a
   guarded source of the dotenv:

   ```bash
   if [ -f /home/pi/.soundsprout/conf ]; then
       set -a
       source /home/pi/.soundsprout/conf
       set +a
   fi
   ```

   `set -a` exports the values so child processes inherit them; the `-f` guard
   keeps `set -euo pipefail` from aborting when the file does not exist yet.

3. **`system/scripts/cmus.sh`** — drop `source /etc/environment` (line 8); it
   uses no variable from it.

4. **`setup.sh`** — seed/validate the chosen source. Runs as root today.
   - ensure `/home/pi/.soundsprout` exists and is `pi:pi`;
   - ensure `/home/pi/.soundsprout/conf` exists, `pi:pi`, `0600`;
   - remove any stale `PJ_BLUETOOTH_DEVICE=` / `PJ_DEFAULT_BRANCH=` lines from
     `/etc/environment` (after backing it up to `/etc/environment.bak`), so
     nothing can shadow the dotenv;
   - if the dotenv has no non-empty `PJ_BLUETOOTH_DEVICE`, print a loud warning
     naming the dotenv path and the portal URL, and **continue** (exit 0).

   Implementation note: `setup.sh` keeps `set -euo pipefail`; the validation
   block must not leak a non-zero status. A missing `/etc/environment` is not
   an error.

5. **Portal** — no production change; `persistAndApply` already writes the
   dotenv + `.asoundrc` + `cmus/rc`. Add a test asserting all three files agree
   after `Activate`.

6. **Docs** —
   - `.env.sample`: single location `/home/pi/.soundsprout/conf`; drop the
     "and/or `/etc/environment`" wording.
   - `README.md` step 16: describe the single location, the portal as the normal
     way to set it, and that `setup.sh` seeds/warns.
   - `README.md` steps 8, 15, 16 and the Web Portal section: make sure every
     `PJ_BLUETOOTH_DEVICE` reference points at the dotenv as the single
     location (step 16 is the only one that currently instructs writing
     `/etc/environment`).
   - `docs/rfid-map-and-bluetooth.md`: update the config-location table
     (`/etc/environment` row) and any `/etc/environment`/`source` references to
     the dotenv.

**Error handling:** a missing MAC keeps `btconnect` FATAL (existing
`ErrMissingDevice`) and `templategen` logging an error; `setup.sh` warns. No
silent-failure path remains.

### Problem B — player CPU + log spam

7. **`server/rfid_reader.py` (new)** — imports only `time`:

   ```python
   import time

   DEFAULT_POLL_INTERVAL = 0.05

   def poll_for_card(reader, poll_interval=DEFAULT_POLL_INTERVAL,
                     sleep=time.sleep):
       while True:
           rfid_id, text = reader.read_no_block()
           if rfid_id:
               return rfid_id, text
           sleep(poll_interval)
   ```

   A separate module keeps `player.py` importable-free (it runs GPIO at import)
   so the helper can be unit-tested with a fake reader.

8. **`server/player.py`** —
   - import and call `poll_for_card(rfid_reader)` in place of
     `rfid_reader.read()`;
   - remove `import warnings` and `warnings.simplefilter("error")`;
   - demote the per-iteration `Loading map file ...` log to `Logger.debug`.

9. **`server/cmus_utils.py`** — demote `Using CMUS socket path` and
   `Completed command` from `Logger.info` to `Logger.debug`. Leave the guarded
   `Sending command` INFO and all error paths unchanged.

10. **`server/peripheral_helpers.py`** — `led_update_loop` calls
    `cmus_status()` once per iteration and unpacks
    `is_playing, is_shuffle, is_repeat`; delete the now-unused
    `music_is_playing` / `music_is_shuffling` / `music_is_repeating` wrappers
    (only the LED loop used them). Poll cadence stays 0.5 s.

### Tests

- **Python** `server/tests/test_rfid_reader.py` — a fake reader returning
  `(None, None)` then a card; a recording `sleep`; assert the card is returned
  and the sleep interval is honored between polls. Mirrors the existing
  `server/tests/` style (plain pytest, no hardware, import from `server/`).
- **Go** `system/soundsprout/portal/bluetooth_test.go` — extend
  `TestBluetoothActivatePersistsAndApplies` (or add a sibling) to read the
  rendered `.asoundrc` and `cmus/rc` and assert they carry the activated MAC
  alongside the dotenv, proving `persistAndApply` keeps every consumer file
  consistent.
- **Verification** (per `AGENTS.md`, not recopied): `python3 -m pytest`,
  `make test`, `golangci-lint run --config system/soundsprout/.golangci.yml
  ./...`, `pre-commit run --all-files`.
- **On-hardware** (needs the user): pair via the portal and confirm dotenv +
  `.asoundrc` + `cmus/rc` all show the new MAC and audio plays; run `setup.sh`
  on an image with no dotenv MAC and confirm it warns without failing; confirm
  the player idles well under 10% and the log stops rotating every few minutes.

## Out of scope

- MFRC522 IRQ wiring / GPIO-interrupt polling.
- `RPi.GPIO`/sysfs GPIO fragility (tracked separately).
- `cmus.sh`'s per-second `sudo supervisorctl status` polling (noted only).
- Any change to btconnect's backoff behavior (already implemented).

## Follow-ups to file in Beads

- On-hardware validation of both fixes (fresh-install warn path, portal speaker
  swap, idle CPU, log rotation).
- `RPi.GPIO` / stale sysfs export fragility (from the `normajean` handoff §4).
