#!/bin/bash

set -xeuo pipefail

echo "Checking for updates at $(date '+%Y-%m-%d %H:%M:%S')..."

cd /home/pi/workspace/pyrfid-jukebox
sudo -u pi git fetch

pipinstall=false
if [ $(git log -n 1 --pretty=format:"%H" -- requirements.txt) != $(git log -n 1 --pretty=format:"%H" origin/main -- requirements.txt) ]; then
    echo "New requirements available."
    pipinstall=true
fi

if [ $(git rev-parse HEAD) != $(git rev-parse @{u}) ]; then
    echo "New version available. Updating..."
    sudo -u pi git reset --hard origin/main
    sudo -u pi git pull

    if [ "$pipinstall" = true ]; then
        echo "Installing requirements"
        sudo -u pi pip3 install -r requirements.txt
    fi

    echo "Restarting pyrfid_jukebox"
    supervisorctl restart pyrfid_jukebox
else
    echo "No updates found."
fi

sudo -u pi touch /home/pi/scripts/setup.sh
if ! diff -q setup.sh /home/pi/scripts/setup.sh; then
    echo "Running setup"
    sudo bash setup.sh
    sudo -u pi cp setup.sh /home/pi/scripts/setup.sh
fi

echo "No updates found. Sleeping for 5 minutes..."
sleep 5m
