import multiprocessing

from rfid_map_store import load_map, save_map_atomic, update_map


def test_load_missing_returns_empty(tmp_path):
    assert load_map(str(tmp_path / "nope.json")) == {}


def test_save_then_load_roundtrip(tmp_path):
    path = str(tmp_path / "map.json")
    save_map_atomic(path, {"1": "/a"})
    assert load_map(path) == {"1": "/a"}


def test_save_writes_indented_json(tmp_path):
    path = tmp_path / "map.json"
    save_map_atomic(str(path), {"1": "/a"})
    assert '    "1": "/a"' in path.read_text()


def test_update_mutates_and_persists(tmp_path):
    path = str(tmp_path / "map.json")
    save_map_atomic(path, {"1": "/a"})

    def mutate(data):
        data["2"] = "/b"
        return "ok"

    assert update_map(path, mutate, lock_path=str(tmp_path / "lock")) == "ok"
    assert load_map(path) == {"1": "/a", "2": "/b"}


def _add_key(args):
    path, lock_path, key = args
    update_map(path, lambda d: d.__setitem__(key, "/x"), lock_path=lock_path)


def test_update_is_safe_across_processes(tmp_path):
    path = str(tmp_path / "map.json")
    lock = str(tmp_path / "lock")
    save_map_atomic(path, {})
    procs = [
        multiprocessing.Process(target=_add_key, args=((path, lock, str(i)),))
        for i in range(5)
    ]
    for p in procs:
        p.start()
    for p in procs:
        p.join()
    assert len(load_map(path)) == 5
