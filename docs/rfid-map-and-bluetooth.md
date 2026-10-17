# RFID Map & Bluetooth Audio — Research Findings

**Where this was saved and why:** This file (`docs/rfid-map-and-bluetooth.md`) was created because the `docs/` directory already exists (holding images and `packages_list.txt`) but contains no markdown notes. This is the natural place for research artifacts.

---

## TL;DR

When an RFID card is scanned, `server/player.py` reads the card's numeric ID, looks it up in `server/rfid_map.json` (a simple `{ "RFID_ID": "/absolute/path/to/folder" }` map). If the ID exists and the folder is found, it stops cmus, clears the queue, adds every `.mp3` file from that folder (sorted alphabetically), and starts playback. Unknown cards are added to the map with an empty-string value (`""`), triggering a red-LED error blink. The map is gitignored, created on first run, and uses PID-based file locking to prevent concurrent instances. Bluetooth audio is routed via BlueALSA → ALSA (`.asoundrc`) → cmus, with a Go service polling `bluetoothctl` (15s cadence, backing off to at most 5 minutes after repeated failures) to auto-reconnect. Template generation (`templategen`) uses `PJ_BLUETOOTH_DEVICE` to produce `.asoundrc` and `cmus/rc` configs.

Individual SoundSprout instance (machine) names are deployment-specific and are deliberately not recorded in this repo.

---

## 1. `rfid_map.json` — Exact Schema

### Location
`server/rfid_map.json` — absolute path resolved at runtime as `os.path.join(script_dir, "rfid_map.json")` where `script_dir = os.path.dirname(os.path.abspath(__file__))` → `/home/pi/workspace/pyrfid-jukebox/server/rfid_map.json` ([`server/player.py:28-29`](server/player.py#L28)).

### Git status
Gitignored via `.gitignore:6` — `server/rfid_map.json` ([.gitignore:6](.gitignore#L6)). **Not present in this checkout.**

### JSON Schema

```json
{
  "<RFID_ID_as_string>": "<absolute_folder_path_or_empty_string>",
  "1061668068229": "/home/pi/workspace/pyrfid-jukebox/music/card-1061668068229",
  "551659120293": "/home/pi/workspace/pyrfid-jukebox/music/card-551659120293",
  "999999999999": ""
}
```

| Field | Type | Description |
|-------|------|-------------|
| Key | `string` | The RFID card's numeric ID, converted to string via `str(rfid_id)` ([`server/player.py:129`](server/player.py#L129)). |
| Value | `string` | Either an **absolute folder path** (e.g., `/home/pi/workspace/pyrfid-jukebox/music/card-123`) or **empty string `""`** as a placeholder for unknown/unmapped cards. |

### Empty-string placeholder behavior
When an unknown card is scanned (ID not in map), the code sets `data[rfid_id] = ""` and writes it back ([`server/player.py:167`](server/player.py#L167)). On the next scan of that card, `data.get(rfid_id, "")` returns `""`, which is falsy, so the `else` branch fires: warning logged, red LEDs blink twice ([`server/player.py:159-164`](server/player.py#L159-L164)). The card remains mapped to `""` until someone manually edits the file to point to a real folder.

### Creation & write format
- **Auto-created** on first run if file doesn't exist: `if not os.path.exists(RFID_TO_MUSIC_MAP): data_to_map(data)` ([`server/player.py:123-124`](server/player.py#L123-L124)).
- Written with `json.dump(data, file, indent=4)` — pretty-printed, 4-space indent ([`server/player.py:68`](server/player.py#L68)).
- Writes happen via `data_to_map()` which opens in `"w"` mode, truncating and rewriting the entire file ([`server/player.py:66-68`](server/player.py#L66-L68)).

---

## 2. Full Read/Lookup/Play Lifecycle

### Step-by-step flow (all in `server/player.py` main loop, lines 113-173)

1. **Wait for cmus** — `while ensure_is_cmus_running()` ([`server/player.py:113`](server/player.py#L113)). This checks for the cmus Unix socket at `$XDG_RUNTIME_DIR/cmus-socket` (defaults to `/home/pi/cmus-socket`) ([`server/cmus_utils.py:73-79`](server/cmus_utils.py#L73-L79)).

2. **Create map file if missing** — `if not os.path.exists(RFID_TO_MUSIC_MAP): data_to_map(data)` ([`server/player.py:123-124`](server/player.py#L123-L124)). Creates an empty `{}`.

3. **Read RFID card** — `rfid_id, text = rfid_reader.read()` then `rfid_id = str(rfid_id)` ([`server/player.py:128-129`](server/player.py#L128-L129)). Uses `mfrc522.SimpleMFRC522`.

4. **Load map** — `with open(RFID_TO_MUSIC_MAP, "r") as file: data = json.load(file)` ([`server/player.py:135-136`](server/player.py#L135-L136)).

5. **Look up folder** — `folder_path = data.get(rfid_id, "")` ([`server/player.py:139`](server/player.py#L139)).
   - If truthy: resolve to absolute path via `os.path.abspath()` ([`server/player.py:141`](server/player.py#L141)).
   - Check `os.path.isdir(folder_path)` ([`server/player.py:146`](server/player.py#L146)).
     - **Folder exists**: blink LEDs, call `execute_cmus_command(QUEUE_AND_PLAY_FOLDER, folder_path)` ([`server/player.py:148-150`](server/player.py#L148-L150)).
     - **Folder missing**: log warning, blink red LEDs twice ([`server/player.py:151-154`](server/player.py#L151-L154)).
   - If falsy (empty string / not in map): log warning, blink red LEDs, set `update_map = True` and `data[rfid_id] = ""` ([`server/player.py:159-167`](server/player.py#L159-L167)).

6. **Write map back** if `update_map` is True — `data_to_map(data)` ([`server/player.py:169-170`](server/player.py#L169-L170)).

### What `QUEUE_AND_PLAY_FOLDER` does (`server/cmus_utils.py:104-112`)

```
1. "player-stop"       → stop current playback
2. "clear"             → clear the cmus queue
3. For each file in the folder (sorted by os.listdir):
     if filename.endswith(".mp3"):
       "add {absolute_path}"   → add to queue
4. "player-next"       → advance to first track
5. "player-play"       → start playback
```

**Only `.mp3` files are played** — the filter is `filename.endswith(".mp3")` ([`server/cmus_utils.py:108`](server/cmus_utils.py#L108)). The comment says "or use a more comprehensive filter" but only `.mp3` is implemented.

### cmus communication
All cmus commands go through a Unix domain socket at `$XDG_RUNTIME_DIR/cmus-socket` (defaulting to `/home/pi/cmus-socket`) ([`server/cmus_utils.py:19-21`](server/cmus_utils.py#L19-L21)). Commands are sent one at a time with `s.send(cmd.encode() + b"\n")`, 50ms sleep between commands, 20s timeout ([`server/cmus_utils.py:34-41`](server/cmus_utils.py#L34-L41)).

---

## 3. Concurrency / Locking Notes for Safe External Editing

### PID-based lock file
- **Lock path**: `/tmp/pyrfid_jukebox.lock` ([`server/player.py:30`](server/player.py#L30)).
- **Mechanism**: On startup, `acquire_lock()` opens the file, reads the PID, checks if that PID is still running via `os.kill(pid, 0)` ([`server/player.py:45-62`](server/player.py#L45-L62)).
- If running → `RuntimeError("Script is already running.")` ([`server/player.py:53-54`](server/player.py#L53-L54)).
- If not running (or file empty) → writes current PID ([`server/player.py:59-61`](server/player.py#L59-L61)).
- **On exit**: closes and deletes the lock file ([`server/player.py:196-197`](server/player.py#L196-L197)).

### Read-modify-write race window
The player **reads** the JSON map ([`server/player.py:135-136`](server/player.py#L135-L136)), optionally **modifies** it (adding unknown card), and **writes** it back ([`server/player.py:169-170`](server/player.py#L169-L170)) all within a single scan cycle. A web portal writing to the file during this window could lose updates or cause JSON corruption.

### Safe external editing strategy
1. The player only **writes** the map when adding an unknown card (setting it to `""`). It never modifies existing mappings.
2. A portal can safely write to `rfid_map.json` between scan cycles. The risk is low because:
   - The write uses `json.dump` with `indent=4` (atomic-ish with `"w"` mode).
   - The player only reads the file once per card scan (not continuously).
3. **However**: `json.load()` will raise on a partially-written file. A portal should write to a temp file then `os.replace()` for atomicity (current code does NOT do this).
4. The supervisor config starts the player with `autostart=false` ([`system/supervisor/conf.d/soundsprout_server.conf:3`](system/supervisor/conf.d/soundsprout_server.conf#L3)) — it's started by `cmus.sh` ([`system/scripts/cmus.sh:72-78`](system/scripts/cmus.sh#L72-L78)).

---

## 4. Music Folder/File Conventions

### On-disk layout
Music lives under `music/` at the project root (`/home/pi/workspace/pyrfid-jukebox/music/`). The `.gitignore` excludes `music/card-*` ([.gitignore:3](.gitignore#L3)), indicating the convention is `music/card-<RFID_ID>/`.

### Historical convention (from `utils/tbd/`)
- `utils/tbd/rfid-folder-creator.py:14` creates directories named `music/card-<id>` where `id` is the raw RFID numeric ID.
- `utils/tbd/play-music.py:18` plays from `music/card-<id>` using `cmus-remote -s -c -f`.

### Current convention (from `server/player.py`)
The map stores **absolute paths** — not relative paths. The value can be anything (e.g., `/home/pi/workspace/pyrfid-jukebox/music/card-123` or any other path). The code does not enforce the `music/card-<id>` pattern; it just checks `os.path.isdir()` ([`server/player.py:146`](server/player.py#L146)).

### Supported file extensions
**Only `.mp3`** — hardcoded in `server/cmus_utils.py:108`: `if filename.endswith(".mp3")`. No FLAC, OGG, WAV, or other formats are supported despite the comment suggesting extensibility.

### Mutagen
`mutagen==1.46.0` is in `requirements.txt` ([`requirements.txt:41`](requirements.txt#L41)) but is **never imported** by any Python file in the repo. It's a dead dependency or was intended for future tag-reading features.

### `sample-cmds.txt`
Contains historical `cmus-remote` command history including an example: `cmus-remote -C "add -q /home/pi/workspace/pyrfid-jukebox/music/card-551659120293"` ([`sample-cmds.txt:38`](sample-cmds.txt#L38)), confirming the `music/card-<RFID_ID>` folder convention was used.

---

## 5. Bluetooth Pairing & ALSA/cmus Config Pipeline

### 5.1 btconnect service (Go)

**Source**: [`system/soundsprout/btconnect/service.go`](system/soundsprout/btconnect/service.go)

- Runs a context-aware loop (default cadence 15 seconds).
- Reads `PJ_BLUETOOTH_DEVICE` env var (required); a missing value is fatal (`ErrMissingDevice`) and stops the process.
- Checks connection: runs `bluetoothctl info <MAC>`, parses output for `"Connected: yes"`.
- If not connected: runs `bluetoothctl connect <MAC>`.
- Connect failures are logged (Warn) and retried with exponential backoff — 15s doubling to a 300s cap, reset on success — instead of terminating the service. A recovery is logged at Info. This prevents an absent speaker from FATALing the service under supervisor (the 2026-10-16 outage: fast exit/restart cycles wedged the Pi's Bluetooth stack; see the runbook in the incident handoff for radio-wedge recovery).

**Supervisor config**: [`system/supervisor/conf.d/btconnect.conf`](system/supervisor/conf.d/btconnect.conf) — runs `/usr/local/bin/soundsprout-server btconnect`, autostart=true, user=pi.

### 5.2 Template generation

**Source**: [`system/soundsprout/templategen/service.go`](system/soundsprout/templategen/service.go)

Generates two config files from templates using `PJ_BLUETOOTH_DEVICE` substitution:

| Template | Destination | Service restarts |
|----------|-------------|-----------------|
| `asoundrc.txt` | `/home/pi/.asoundrc` | `sudo supervisorctl restart btconnect`, `sudo alsactl restore` |
| `config-cmus-rc.txt` | `/home/pi/.config/cmus/rc` | `sudo supervisorctl restart cmus_manager` |

([`service.go:34-49`](system/soundsprout/templategen/service.go#L34-L49))

**asoundrc template** ([`templates/asoundrc.txt`](system/soundsprout/templategen/templates/asoundrc.txt)):
```
defaults.bluealsa.device "{{.PJ_BLUETOOTH_DEVICE}}"
pcm.!default {
    type bluealsa
    device "{{.PJ_BLUETOOTH_DEVICE}}"
    profile "a2dp"
}
```

**cmus rc template** ([`templates/config-cmus-rc.txt`](system/soundsprout/templategen/templates/config-cmus-rc.txt)):
```
set dsp.alsa.device=bluealsa:DEV={{.PJ_BLUETOOTH_DEVICE}},PROFILE=a2dp,SRV=org.bluealsa
```

**runner.sh**: The `templategen` command generates a `runner.sh` script that moves generated files to their destinations, chowns to pi, and restarts services — but only if content has changed ([`service.go:70-129`](system/soundsprout/templategen/service.go#L70-L129)).

### 5.3 cmus manager (`cmus.sh`)

**Source**: [`system/scripts/cmus.sh`](system/scripts/cmus.sh)

- Runs in a loop, managed by supervisor ([`cmus_manager.conf`](system/supervisor/conf.d/cmus_manager.conf)).
- Starts cmus via `screen -dmS cmus /usr/bin/cmus --listen $socket_file` ([`cmus.sh:43`](system/scripts/cmus.sh#L43)).
- Socket at `$XDG_RUNTIME_DIR/cmus-socket` → `/home/pi/cmus-socket` ([`cmus.sh:11`](system/scripts/cmus.sh#L11)).
- Detects if cmus autosave has changed and restarts cmus if so ([`cmus.sh:23-31`](system/scripts/cmus.sh#L23-L31)).
- Auto-starts `soundsprout_server` if not running ([`cmus.sh:72-78`](system/scripts/cmus.sh#L72-L78)).

### 5.4 Full Bluetooth audio pipeline

```
Bluetooth Speaker (A2DP)
    ↑
BlueALSA (bluealsa systemd service)
    ↑
ALSA (.asoundrc → bluealsa PCM plugin)
    ↑
cmus (--listen socket, output_plugin=alsa, dsp.alsa.device=bluealsa:DEV=...)
    ↑
player.py → cmus socket commands (add .mp3 files, player-play)
```

### 5.5 Upstream Bluetooth tooling

The repo's README documents `bluetoothctl` pairing steps ([`README.md:93-109`](README.md#L93-L109)):
1. `bluetoothctl` → enter the interactive tool
2. `power on`
3. `scan on`
4. `agent on`
5. `pair <MAC>`
6. `trust <MAC>`
7. `connect <MAC>`

These are BlueZ's `bluetoothctl` commands. See [BlueZ documentation](https://git.kernel.org/pub/scm/bluetooth/bluez.git/tree/README) for `bluetoothctl` behavior.

BlueALSA bridges ALSA to Bluetooth. See [bluez-alsa README](https://github.com/arkq/bluez-alsa) and [INSTALL.md](https://github.com/arkq/bluez-alsa/blob/master/INSTALL.md) for details on the ALSA plugin.

**Note**: The repo does NOT implement `bluetoothctl pair/trust` programmatically. Pairing must be done manually via `bluetoothctl`. The Go `btconnect` service only handles `info`/`connect` (reconnection). For initial pairing, the user must run the interactive steps above.

---

## 6. Web Server / HTTP Endpoints

**No web server is currently running.** Despite `Flask==3.0.0` and `aiohttp==3.8.4` being in `requirements.txt` ([`requirements.txt:3,26`](requirements.txt#L3)), grep across all `.py` files shows **zero imports** of flask, aiohttp, or any HTTP server framework. No `app.run()`, no route decorators, no HTTP listeners.

Flask is listed in the README's "Parental Control Module" feature as "Remote management capabilities allowing updates and maintenance via SSH or a web interface" ([`README.md:15`](README.md#L15)), but this has not been implemented yet.

---

## 7. Python Version / venv / Config Locations

### Python environment
- venv created at `/home/pi/workspace/pyrfid-jukebox/env` ([`README.md:283`](README.md#L283)): `python3 -m venv --system-site-packages env`
- Supervisor runs: `/home/pi/workspace/pyrfid-jukebox/env/bin/python -u /home/pi/workspace/pyrfid-jukebox/server/player.py` ([`soundsprout_server.conf:4`](system/supervisor/conf.d/soundsprout_server.conf#L4))
- Python version: Not pinned in requirements.txt. The setup uses `python3-venv` from the system apt ([`README.md:281`](README.md#L281)). On Raspberry Pi OS Bullseye this is Python 3.9.

### Config file locations (on live device)
| File | Purpose |
|------|---------|
| `/home/pi/workspace/pyrfid-jukebox/server/rfid_map.json` | RFID-to-folder mapping |
| `/home/pi/.asoundrc` | ALSA Bluetooth audio config |
| `/home/pi/.config/cmus/rc` | cmus runtime config (Bluetooth device) |
| `/home/pi/.config/cmus/autosave` | cmus state (persisted settings) |
| `/home/pi/.soundsprout/conf` | dotenv for `PJ_BLUETOOTH_DEVICE` (loaded by Go CLI) |
| `/tmp/pyrfid_jukebox.lock` | Player process lock |
| `/home/pi/cmus-socket` | cmus Unix domain socket |
| `/etc/environment` | System-wide env vars including `PJ_BLUETOOTH_DEVICE` |
| `/home/pi/logs/` | All supervisor log files |

---

## 8. Primary Sources

| Source | Path |
|--------|------|
| RFID player main loop, map read/write, locking | `server/player.py:1-201` |
| cmus socket communication, QUEUE_AND_PLAY_FOLDER, .mp3 filter | `server/cmus_utils.py:1-130` |
| GPIO buttons, LEDs | `server/peripheral_helpers.py:1-197` |
| Logger | `server/logger.py:1-30` |
| btconnect Go service | `system/soundsprout/btconnect/service.go:1-106` |
| CLI main (btconnect, templategen commands) | `system/soundsprout/cmd/main.go:1-95` |
| Template generation service | `system/soundsprout/templategen/service.go:1-207` |
| asoundrc template | `system/soundsprout/templategen/templates/asoundrc.txt:1-17` |
| cmus rc template | `system/soundsprout/templategen/templates/config-cmus-rc.txt:1` |
| soundsprout_server supervisor config | `system/supervisor/conf.d/soundsprout_server.conf:1-13` |
| btconnect supervisor config | `system/supervisor/conf.d/btconnect.conf:1-12` |
| cmus_manager supervisor config | `system/supervisor/conf.d/cmus_manager.conf:1-13` |
| cmus.sh script | `system/scripts/cmus.sh:1-82` |
| gitpull.sh script | `system/scripts/gitpull.sh:1-69` |
| prepare_logs.sh script | `system/scripts/prepare_logs.sh:1-66` |
| setup.sh script | `setup.sh:1-21` |
| .gitignore | `.gitignore:1-12` |
| requirements.txt | `requirements.txt:1-82` |
| README.md (Bluetooth, setup docs) | `README.md:1-418` |
| rfid-folder-creator.py (historical) | `utils/tbd/rfid-folder-creator.py:1-18` |
| play-music.py (historical) | `utils/tbd/play-music.py:1-28` |
| sample-cmds.txt | `sample-cmds.txt:1-38` |

### Upstream documentation
- **BlueZ bluetoothctl**: <https://git.kernel.org/pub/scm/bluetooth/bluez.git/tree/README>
- **BlueALSA (bluez-alsa)**: <https://github.com/arkq/bluez-alsa>
- **BlueALSA INSTALL**: <https://github.com/arkq/bluez-alsa/blob/master/INSTALL.md>
- **cmus**: `man cmus-remote` (local man page)

---

## 9. Follow-ups

Tracked outside this document:

- **Python version not pinned** — issue #17.
- **Initial Bluetooth pairing not automated** — issue #18.
- **Mutagen unused** — issue #15.
- **Only `.mp3` files are played** — issue #16.
- **README/repo config drift** (`btconnect.sh`, `system/home/.asoundrc`, empty `.env.sample`) — `docs/handoff-repo-cleanup.md`.

The `rfid_map.json` writer is not atomic (`data_to_map()` uses `open("w")` directly,
[`server/player.py:66-68`](server/player.py#L66)); the web-portal design should use
`os.replace()` for safe writes.
