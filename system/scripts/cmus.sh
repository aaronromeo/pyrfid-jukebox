#!/bin/bash

set -xeuo pipefail

if ! screen -list | grep -q "cmus"; then
    echo "Starting cmus..."
    /usr/bin/screen -dmS cmus /usr/bin/cmus
else
    echo "cmus already running."
    sleep 60 * 15
fi
