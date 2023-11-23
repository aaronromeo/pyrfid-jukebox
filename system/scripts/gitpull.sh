#!/bin/bash

set -xeuo pipefail

echo
echo "$(date '+%Y-%m-%d %H:%M:%S') - Script started"

cd /home/pi/workspace/pyrfid-jukebox
echo "Current directory: $(pwd)"
echo "Listing remote branches:"
sudo -u pi git branch -r

echo "Fetching from remote..."
sudo -u pi git fetch

pipinstall=false
echo "Checking for updates in requirements.txt..."
if ! sudo -u pi git diff --quiet origin/main...HEAD -- requirements.txt; then
    echo "New requirements found in requirements.txt"
    pipinstall=true
else
    echo "No new requirements in requirements.txt"
fi

repodiffs=false
if [ $(sudo -u pi git rev-parse HEAD) != $(sudo -u pi git rev-parse @{u}) ]; then
    echo "New version available. Updating..."
    sudo -u pi git reset --hard origin/main
    sudo -u pi git pull
    repodiffs=true
else
    echo "No updates found."
fi

sudo -u pi touch /home/pi/scripts/setup.sh
if ! diff -q setup.sh /home/pi/scripts/setup.sh; then
    echo "Running setup"
    sudo bash setup.sh
    sudo -u pi cp setup.sh /home/pi/scripts/setup.sh
fi

if [ "$pipinstall" = true ]; then
    echo "Installing requirements"
    sudo -u pi pip3 install -r requirements.txt
fi

if [ "$repodiffs" = true ]; then
    echo "Restarting pyrfid_jukebox"
    supervisorctl restart pyrfid_jukebox
fi

echo "Sleeping for 5 minutes..."
sleep 5m
