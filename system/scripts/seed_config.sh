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
