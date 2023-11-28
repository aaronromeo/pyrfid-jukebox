#!/bin/bash

set -xeuo pipefail

echo
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

if ! test -f $XDG_RUNTIME_DIR && ([ $screen_exit_status -ne 0 ] || [ -z "$screen_session" ]); then
    echo "$(date '+%Y-%m-%d %H:%M:%S') Socket file does not exist but screen is active."
    set +e
    screen -S cmus -X quit # Kill the screen
    set -e  # Re-enable 'exit on error'
fi

if [ $screen_exit_status -ne 0 ] || [ -z "$screen_session" ]; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') Starting cmus..."
    /usr/bin/screen -dmS cmus /usr/bin/cmus 2> /home/pi/logs/process_cmus_error.log > /home/pi/logs/process_cmus_output.log
else
    echo "$(date '+%Y-%m-%d %H:%M:%S') cmus already running."
fi

set +e
# Debugging possible file system usage of the CMUS lock file
lsof -V $XDG_RUNTIME_DIR/cmus-socket
set -e  # Re-enable 'exit on error'

status=$(sudo supervisorctl status pyrfid_jukebox | awk '{print $2}')
if [[ $status != "RUNNING" && $status != "STARTING" ]]; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') pyrfid_jukebox is not running or starting. Starting it now..."
    sudo supervisorctl start pyrfid_jukebox
else
    echo "$(date '+%Y-%m-%d %H:%M:%S') pyrfid_jukebox is already running or starting."
fi
