#!/bin/bash

set -xeuo pipefail

echo "Checking for updates at $(date '+%Y-%m-%d %H:%M:%S')..."

cd /home/pi/workspace/pyrfid-jukebox
sudo -u pi git fetch

if [ $(git rev-parse HEAD) != $(git rev-parse @{u}) ]; then
    echo "New version available. Updating..."
    sudo -u pi git pull

    echo "Installing requirements"
    sudo -u pi pip3 install -r requirements.txt

    echo "Running setup"
    bash setup.sh

    echo "Restarting pyrfid_jukebox"
    supervisorctl restart pyrfid_jukebox
else
    echo "No updates found. Sleeping..."
    sleep 60 * 5
fi
