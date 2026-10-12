# SoundSprout Web Portal — Design

- **Status:** Draft for review
- **Date:** 2026-10-12
- **Related:** `docs/rfid-map-and-bluetooth.md` (research), `docs/handoff-repo-cleanup.md`
- **Issues:** #15 (Mutagen), #16 (mp3-only), #17 (Python version), #18 (Bluetooth pairing), #19 (off-device simulation mode)

## 1. Context

SoundSprout maps RFID cards to music folders via `server/rfid_map.json`, a flat
`{ "<card_id>": "<absolute_folder_path>" }` map owned by the Python `player.py`.
When a card is scanned, the player looks up its ID, and if the mapped folder exists it
queues that folder's `.mp3` files into `cmus`. Unrecognized cards are recorded with an
empty-string value (`"<card_id>": ""`) so they appear as "unassigned". Bluetooth audio
is routed BlueALSA → ALSA (`/home/pi/.asoundrc`) → cmus, with the Go `btconnect`
service maintaining the connection and `templategen` rendering the audio configs from
`PJ_BLUETOOTH_DEVICE`.

Today there is **no web interface** (Flask is listed in `requirements.txt` but unused),
so editing associations and pairing Bluetooth requires SSH. This document specifies a
web portal to do both.

See `docs/rfid-map-and-bluetooth.md` for the full mechanism with source citations.

## 2. Goals / Non-goals

### Goals
1. **Cards:** list mappings, assign unassigned cards via a folder picker over `music/`, edit/repoint, remove.
2. **Bluetooth:** show current status (connected MAC), full pairing (scan → pair/trust/connect), switch active device, reconnect.
3. Add the portal as a compiled Go `serve` subcommand of the existing `soundsprout-server` binary.
4. Harden the shared `rfid_map.json` write path (atomic write + cross-language lock) in `player.py`.

### Non-goals (v1)
Music upload/ripping, library management, live "scan a card now", manual card-ID entry,
now-playing/playback controls, reboot controls, authentication, off-device simulation
mode (issue #19).

## 3. Key Decisions

| # | Decision | Choice |
|---|----------|--------|
| D1 | How a new card→music association is created | Assign from already-seen unknown cards (the `""` entries) |
| D2 | How the music target is chosen | Folder picker over the project `music/` directory |
| D3 | Bluetooth scope | Full pairing **and** status/reconnect (supersedes #18's pairing guts) |
| D4 | Access control | LAN-only, no login |
| D5 | Privileged steps | Portal runs as `pi` with narrowly-scoped passwordless sudo |
| D6 | Implementation language | Go, as a new `soundsprout-server serve` subcommand |
| D7 | Active-MAC persistence | Canonical `pi`-owned dotenv at `/home/pi/.soundsprout/conf`; fix the loader to that absolute path |
| D8 | Python testing | Add `pytest` scoped to the extracted map-store module |

## 4. Architecture

A new `serve` subcommand on the existing `soundsprout-server` binary runs a small HTTP
portal. It is supervised as a new program (`portal.conf`, user `pi`, autostart),
reachable on the LAN with no auth. It owns no hardware and no state — it reads/writes
`server/rfid_map.json`, shells out to `bluetoothctl`, writes the `pi`-owned audio config
files, and calls a small set of `sudo` restart commands.

```text
Phone/Laptop ──HTTP──▶ soundsprout-server serve (as pi, :8080)
                          ├─ reads/writes  server/rfid_map.json        (atomic + flock)
                          ├─ lists folders under  music/
                          ├─ exec bluetoothctl  (status/scan/pair/trust/connect)
                          ├─ writes  ~/.asoundrc, ~/.config/cmus/rc     (reuse templategen templates)
                          ├─ writes  /home/pi/.soundsprout/conf          (active MAC, preserving other keys)
                          └─ sudo supervisorctl restart {btconnect,cmus_manager}; sudo alsactl restore
```

New/changed files:

- `system/soundsprout/portal/` — new package (logic + HTTP + embedded templates/tests)
- `system/soundsprout/btconnect/service.go` — extract reusable `info`→`connect` logic
- `system/soundsprout/cmd/main.go` — register `serve`; fix dotenv path to absolute
- `system/supervisor/conf.d/portal.conf` — new supervised program
- `setup.sh` — install `NOPASSWD` sudoers file for the whitelisted commands
- `system/scripts/prepare_logs.sh` — add portal log files
- `server/rfid_map_store.py` — extracted map I/O (atomic write + lock)
- `server/player.py` — use `rfid_map_store`
- `server/tests/test_rfid_map_store.py` — pytest tests
- `requirements.txt` — add `pytest` (pinned)
- `.github/workflows/testing.yml` — run pytest
- `README.md`, `AGENTS.md` — document the portal

**Portability note:** all paths are injected into the portal modules (so unit tests use
`t.TempDir()`), but v1 exposes **no CLI override flags**; the canonical Pi paths are the
defaults. Off-device simulation (`--runner=fake`, path overrides) is deferred to #19.

## 5. Components & Seams

One `portal` package split into deep, individually testable modules behind a single
injected command seam (mirroring `btconnect`).

```go
// The one impure seam — everything external goes through this.
type CommandRunner interface {
    Output(name string, args ...string) (string, error) // stdout for parsing
    Run(name string, args ...string) error              // side-effecting commands
}
// prod: execRunner (os/exec)   test: fakeRunner (canned bluetoothctl output)
```

```text
system/soundsprout/portal/
  service.go    Service: wires deps, registers routes on http.ServeMux, runs server
  rfidmap.go    MapStore: Load()/Save() (atomic temp+rename + flock); Assign/Remove/Unassigned
  music.go      Library: Folders() over music/; Resolve(name) with music-dir confinement
  bluetooth.go  Bluetooth: Status/PairedDevices/Scan/Pair/Connect; pure parsers
  config.go     AudioConfig: render .asoundrc + cmus/rc (reuse templategen), whitelisted restarts; Apply(mac)
  settings.go   Settings: Get()/SetActiveMAC() for /home/pi/.soundsprout/conf
  http.go       handlers: translate HTTP <-> modules (no business logic)
  templates/    embedded HTML (go:embed), server-rendered, no JS build
  *_test.go     table tests + golden files + fakeRunner
```

Module contracts:

- **`MapStore`** — the only writer of `server/rfid_map.json` (besides `player.py`). `Save`
  is atomic (`os.WriteFile` to a temp file in the same dir, then `os.Rename`) and takes an
  exclusive `flock` for the read-modify-write. `Load` treats a missing file as empty.
- **`Library`** — lists immediate subfolders of `music/` (the player only plays `.mp3`s
  directly in the chosen folder). `Resolve` rejects paths that escape `music/`.
- **`Bluetooth`** — typed wrapper over `bluetoothctl`; all output parsing is in pure
  functions tested without a radio.
- **`AudioConfig`** — calls the existing
  `templategen.GenerateTemplate("templates/asoundrc.txt"|"config-cmus-rc.txt", {PJ_BLUETOOTH_DEVICE: mac})`
  so there is one source of truth for those files, writes the `pi`-owned destinations,
  and invokes only whitelisted restart commands.
- **`Settings`** — persists the active MAC (see §7).
- **`http.go`** — request parsing, validation, status codes, template rendering; delegates
  everything else.

## 6. Data Model, HTTP API, UI

**On disk (unchanged):** `rfid_map.json` stays a flat `{ "<card_id>": "<abs_path>" }`.
The portal derives views from it — *assigned* = non-empty value, *unassigned* = `""` —
plus folder metadata (exists? direct `.mp3` count) computed at render time.

**HTTP surface** (server-rendered HTML with plain forms, POST/Redirect/GET; no JS
framework, no build step):

```text
GET  /cards                  page: mappings table + unassigned cards + assign form
POST /cards/assign           {card_id, folder} -> set mapping
POST /cards/remove           {card_id}         -> delete key (reappears as unassigned if scanned again)
GET  /bluetooth              page: status + known devices + scan + pair form
POST /bluetooth/scan         discover ~5s, then redirect back with results
POST /bluetooth/pair         {mac}  pair+trust+connect, persist, apply, restart
POST /bluetooth/connect      {mac}  switch active output (persist+apply+restart)
POST /bluetooth/reconnect    reconnect the configured device
```

**UI shape:**

- **Cards tab** — table of every card → target folder, with a "missing" badge when the
  folder no longer exists and the count of playable `.mp3`s; a prominent *Unassigned
  cards* section (the `""` entries) each with a folder `<select>` populated from
  `music/`; per-row *Edit* and *Remove*. Assigning an unassigned card and repointing an
  assigned one are the same underlying write.
- **Bluetooth tab** — adapter on/off, current configured MAC, `Connected: yes/no`; a
  **Scan** button; discovered devices each with *Pair*; known devices each with
  *Connect* (make active); a *Reconnect* for the current device.

**Worked flows:**

1. *Assign card* → validate `card_id` is a known key and `folder` resolves inside
   `music/` and is a directory → `MapStore.Load`, set, `Save` (atomic + locked) →
   redirect `/cards`.
2. *Pair device* → validate MAC format → `Bluetooth.Pair` (pair→trust→connect) →
   `Settings.SetActiveMAC` → `AudioConfig.Apply` (render both templates, write
   `pi`-owned files, run whitelisted restarts) → redirect `/bluetooth`.

## 7. Bluetooth Apply Pipeline, Privileges, Settings

**Bluetooth operations** (all via `bluetoothctl`, no shell):

```text
status       info <MAC>   -> parse Name / Paired / Trusted / "Connected: yes"
adapter      show         -> "Powered: yes"
known        devices      -> lines "Device <MAC> <Name>"
scan         scan on  (ctx timeout N)  scan off   -> then `devices`   [parse pure]
pair         pair <MAC> ; trust <MAC> ; connect <MAC>
connect      connect <MAC>                     (switch active output)
reconnect    info <MAC> -> if not connected, connect <MAC>   (btconnect logic)
```

Scan/pair are synchronous form POSTs (~5s for scan); the request blocks, then redirects.
Pairing a headless A2DP sink needs no PIN; if a PIN is required it fails and surfaces as
`?err=`. The `info`→`connect` logic is extracted from `btconnect` so the portal's
*Reconnect* and the `btconnect` daemon share one implementation.

**Config apply pipeline** (mirrors `runner.sh` order):

1. Render `asoundrc.txt` + `config-cmus-rc.txt` via `templategen.GenerateTemplate` with
   `{PJ_BLUETOOTH_DEVICE: mac}`.
2. Write `~/.asoundrc` and `~/.config/cmus/rc` (ensure `~/.config/cmus/` exists) —
   `pi`-owned, so no sudo.
3. `sudo supervisorctl restart btconnect` → `sudo supervisorctl restart cmus_manager` →
   `sudo alsactl restore`.

**Privilege model (D5).** Portal runs as `pi`. `setup.sh` installs
`/etc/sudoers.d/soundsprout-portal` (validated with `visudo -cf`) granting `NOPASSWD` for
exactly:

```text
/usr/bin/supervisorctl restart btconnect
/usr/bin/supervisorctl restart cmus_manager
/usr/sbin/alsactl restore
```

No wildcards, no user input in any command.

**`Settings` persistence (D7).** The canonical store of `PJ_BLUETOOTH_DEVICE` is the
`pi`-owned dotenv `/home/pi/.soundsprout/conf`. The portal writes it directly (no sudo),
preserving other keys (e.g. `PJ_DEFAULT_BRANCH`), atomically + under its own
`flock` (`/tmp/pyrfid_jukebox.settings.lock`). `cmd/main.go`'s dotenv load is changed from the CWD-relative
`home/pi/.soundsprout/conf` to the absolute `/home/pi/.soundsprout/conf`, matching what
`README.md` and `.env.sample` now document. `/etc/environment` remains for the shell
scripts (`gitpull.sh`, `cmus.sh`) that source it, but is not the portal's target.

## 8. Python Hardening (in scope)

Currently `player.py`'s `data_to_map()` writes non-atomically (`open("w")`,
`server/player.py:66-68`), and its read-modify-write of the map can be clobbered by a
concurrent portal write (lost update) or, conversely, the portal can read a partial file.

- Extract map I/O into `server/rfid_map_store.py`:
  - `load_map(path) -> dict` — missing file → empty dict.
  - `save_map_atomic(path, data)` — temp file in the same dir, `fsync`, `os.replace`.
  - `update_map(path, mutate_fn)` — acquire an exclusive `flock` on a dedicated lock file
    for the whole read-modify-write.
- Lock file: `/tmp/pyrfid_jukebox.map.lock` (separate from the process lock
  `/tmp/pyrfid_jukebox.lock`). The **same path and `flock(LOCK_EX)` semantics** are used
  by the Go `MapStore`. This is a documented cross-language contract.
- `player.py` switches its load to `load_map` and its unknown-card write to `update_map`.
  Reads remain lock-free (atomic replace guarantees a complete file); only writers lock.

## 9. Testing

- **Go (hermetic, runs in current CI):** `portal` tests use `fakeRunner` + `t.TempDir()`.
  Table tests for the `bluetoothctl` parsers, map load/save/assign/remove, and the
  `music/` path-confinement guard; golden files for rendered `.asoundrc` / `cmus/rc`
  (same `testdata/baselines` style as `templategen`). Feeds the existing coverage
  artifact.
- **Python (new, scoped):** `server/tests/test_rfid_map_store.py` via `pytest` (pinned in
  `requirements.txt`), plus a CI step. Covers atomicity and the lock.
- **Cross-language lock contract:** the locked-file path and `flock(LOCK_EX)` held across
  read-modify-write are documented here; Go tests and Python tests each assert their side
  honors it. A true two-runtime integration test is out of scope.
- **On-device acceptance (Pi, manual):** a short smoke checklist (assign a card and play
  it; pair a speaker and confirm audio).

## 10. Error Handling

- 400 on validation (bad MAC, folder outside `music/`, unknown card id); 4xx/5xx on command
  failure; surfaced via `?err=`.
- Bluetooth commands run under a context deadline; pairing failure leaves state unchanged
  (MAC is persisted **only after** pair+trust+connect succeed).
- Apply is idempotent (re-running reconnect/apply is safe). If a config write succeeds but
  a restart fails, report and allow retry — never leave the map half-written.
- Lock acquisition has a timeout; on timeout the writer logs and skips (no corruption).

## 11. Security

- No auth, LAN-only — documented as the accepted risk (D4).
- No shell anywhere: `exec.Command(name, args…)` only; strict MAC regex
  `^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`; folder path confined to `music/`; `card_id` must
  already be a key; no user input reaches a shell or an unchecked path.
- Narrow `NOPASSWD` sudoers whitelist; no wildcards.
- Cheap CSRF mitigation: require same-host `Origin`/`Referer` on POSTs.

## 12. Deployment

- `system/supervisor/conf.d/portal.conf`: user `pi`, autostart, JSON logs to stdout →
  `/home/pi/logs/portal.{out,err}.log`.
- `setup.sh` installs the sudoers file (validated with `visudo -cf`) and copies the conf
  (existing `cp system/supervisor/conf.d/*` already covers it).
- `prepare_logs.sh` creates the portal log files.
- `README.md` + `AGENTS.md` document the portal, the new subcommand, and how to run tests.

## 13. Out of Scope / Follow-ups

- Off-device simulation mode (`--runner=fake`, path overrides) — issue #19.
- Automated pairing as a standalone CLI — issue #18 (portal supersedes its pairing guts).
- Python version pinning — issue #17.
- `.mp3`-only playback — issue #16; unused `mutagen` — issue #15.

## 14. Coordination Notes

- `README.md` / `.env.sample` were updated by a prior task (commit `4de87ed`) to document
  `/home/pi/.soundsprout/conf` as the Go dotenv path; this spec's loader fix (D7) makes the
  code match. No further README churn for that item is expected beyond adding portal docs.
- `docs/handoff-repo-cleanup.md` is resolved/committed; do not recreate it.

## 15. Validation Checklist (before implementation)

- [ ] Confirm `templategen.GenerateTemplate` is importable/reusable from the `portal` package.
- [ ] Confirm the `btconnect` `info`→`connect` extraction keeps existing tests green.
- [ ] Confirm current `player.py` uses `data_to_map` at the two call sites and that swapping
      to `rfid_map_store` is mechanical.
- [ ] Confirm supervisor `environment=`/dotenv behavior so `btconnect` sees the new MAC after restart.
- [ ] Confirm `setup.sh` runs as root and is the right place for the sudoers file.
