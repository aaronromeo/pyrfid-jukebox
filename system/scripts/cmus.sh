#!/bin/bash

set -xeuo pipefail

export XDG_RUNTIME_DIR="/run/user/$(id -u pi)"

if ! grep -q "XDG_RUNTIME_DIR" ~/.bashrc; then
  echo 'export XDG_RUNTIME_DIR="/run/user/$(id -u pi)"' >> ~/.bashrc
fi

if ! screen -list | grep -q "cmus"; then
    echo "Starting cmus..."
    /usr/bin/screen -dmS cmus /usr/bin/cmus
else
    echo "cmus already running. Sleeping for 15 minutes..."
    sleep 15m
fi
