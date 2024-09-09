#!/bin/bash

set -xeuo pipefail

echo
echo "$(date '+%Y-%m-%d %H:%M:%S') - Script started"

export XDG_RUNTIME_DIR="/home/pi"
socket_file=$XDG_RUNTIME_DIR/cmus-socket

if ! grep -q "XDG_RUNTIME_DIR" ~/.bashrc; then
  echo "export XDG_RUNTIME_DIR=\"$XDG_RUNTIME_DIR\"" >> ~/.bashrc
fi
while true; do
    # Check if the cmus screen session exists
    set +e
    screen_session=$(screen -list | grep "cmus")
    screen_exit_status=$?
    set -e  # Re-enable 'exit on error'

    if ! diff /home/pi/.config/cmus/autosave /home/pi/workspace/pyrfid-jukebox/system/home/.config/cmus/autosave >/dev/null; then
        echo "$(date '+%Y-%m-%d %H:%M:%S') CMUS autosave has changed in repo. Copying over system config..."
        if [ -n "$screen_session" ]; then
            set +e
            screen -S cmus -X quit # Kill the screen
            screen_session=""
            set -e  # Re-enable 'exit on error'
        fi
        cp /home/pi/workspace/pyrfid-jukebox/system/home/.config/cmus/autosave /home/pi/.config/cmus/autosave
    fi

    if ! test -S $socket_file && [ -n "$screen_session" ]; then
        echo "$(date '+%Y-%m-%d %H:%M:%S') $socket_file does not exist but screen is active."
        set +e
        screen -S cmus -X quit # Kill the screen
        screen_session=""
        set -e  # Re-enable 'exit on error'
    fi

    if [ $screen_exit_status -ne 0 ] || [ -z "$screen_session" ]; then
        echo "$(date '+%Y-%m-%d %H:%M:%S') Starting cmus..."
        /usr/bin/screen -dmS cmus /usr/bin/cmus --listen $socket_file 2> /home/pi/logs/process_cmus_error.log > /home/pi/logs/process_cmus_output.log

        sleep 5  # Wait a bit for CMUS to start and create the socket file

        checkCount=60
        while [ $checkCount -gt 0 ]; do
            sync
            if test -S "$socket_file"; then
                echo "$socket_file has been created"
                break
            else
                echo "$socket_file is still gone, checking in 2 seconds"
                checkCount=$((checkCount - 1))
            fi
        done
    else
        echo "$(date '+%Y-%m-%d %H:%M:%S') cmus already running."
    fi

    set +e
    # Debugging possible file system usage of the CMUS lock file
    if ! test -S $socket_file; then
        echo "$socket_file is still gone"
    else
        lsof -V $socket_file
    fi
    set -e  # Re-enable 'exit on error'

    set +e
    status=$(sudo supervisorctl status pyrfid_jukebox | awk '{print $2}')
    set -e  # Re-enable 'exit on error'
    if [[ $status != "RUNNING" && $status != "STARTING" ]]; then
        echo "$(date '+%Y-%m-%d %H:%M:%S') pyrfid_jukebox is not running or starting. Starting it now..."
        sudo supervisorctl start pyrfid_jukebox
    else
        echo "$(date '+%Y-%m-%d %H:%M:%S') pyrfid_jukebox is already running or starting."
    fi

    sleep 1 # Avoiding the `Exited too quickly (process log may have details)` error
done
