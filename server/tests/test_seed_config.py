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
    (conf_dir / "conf").write_text("PJ_BLUETOOTH_DEVICE=XX:XX:XX:XX:XX:XX\n")

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
