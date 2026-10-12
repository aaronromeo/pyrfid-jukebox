"""Atomic, lock-protected I/O for the RFID-to-music map.

The lock file and flock(LOCK_EX) semantics are shared with the Go portal
(see system/soundsprout/portal/rfidmap.go). Do not change the lock path
without changing the Go side too.
"""

import fcntl
import json
import os
import tempfile

MAP_LOCK = "/tmp/pyrfid_jukebox.map.lock"


def load_map(path):
    if not os.path.exists(path):
        return {}
    with open(path) as handle:
        contents = handle.read()
    if not contents:
        return {}
    return json.loads(contents)


def save_map_atomic(path, data):
    directory = os.path.dirname(os.path.abspath(path))
    os.makedirs(directory, exist_ok=True)
    fd, tmp = tempfile.mkstemp(
        dir=directory, prefix=".rfid_map-", suffix=".tmp"
    )
    try:
        with os.fdopen(fd, "w") as handle:
            json.dump(data, handle, indent=4)
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(tmp, path)
    except BaseException:
        try:
            os.unlink(tmp)
        except OSError:
            pass
        raise


def update_map(path, mutate, lock_path=MAP_LOCK):
    with open(lock_path, "a+") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        try:
            data = load_map(path)
            result = mutate(data)
            save_map_atomic(path, data)
        finally:
            fcntl.flock(lock, fcntl.LOCK_UN)
    return result
