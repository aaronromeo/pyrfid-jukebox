#!/bin/bash

set -xeuo pipefail

echo "$(date '+%Y-%m-%d %H:%M:%S') - Script started"

export XDG_RUNTIME_DIR="/run/user/$(id -u pi)"

if ! grep -q "XDG_RUNTIME_DIR" ~/.bashrc; then
  echo 'export XDG_RUNTIME_DIR="/run/user/$(id -u pi)"' >> ~/.bashrc
fi

# Check if the cmus screen session exists
set +e
screen_session=$(screen -list | grep "cmus")
screen_exit_status=$?
set -e  # Re-enable 'exit on error'

if [ $screen_exit_status -ne 0 ] || [ -z "$screen_session" ]; then
    echo "Starting cmus..."
    /usr/bin/screen -dmS cmus /usr/bin/cmus
else
    echo "cmus already running. Sleeping for 15 minutes..."
    sleep 15m
fi
