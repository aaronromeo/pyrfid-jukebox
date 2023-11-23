#!/bin/bash

echo "$(date '+%Y-%m-%d %H:%M:%S') - Script started"

# Define the lock file and the Python script paths
LOCK_FILE="/tmp/pyrfid_jukebox.lock"
PYTHON_SCRIPT="/home/pi/workspace/pyrfid-jukebox/server/player.py"

# Function to start the Python script
start_script() {
    /home/pi/workspace/pyrfid-jukebox/env/bin/python "$PYTHON_SCRIPT"
}

# Check if the Python script is already running
if pgrep -f "$PYTHON_SCRIPT" > /dev/null; then
    echo "Python script is already running."
else
    echo "Python script is not running."

    # If the lock file exists, remove it
    if [ -f "$LOCK_FILE" ]; then
        echo "Removing stale lock file."
        rm -f "$LOCK_FILE"
    fi

    # Start the Python script
    echo "Starting Python script."
    start_script
fi
