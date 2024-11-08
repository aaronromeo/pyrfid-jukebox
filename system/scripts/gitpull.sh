#!/bin/bash

set -euo pipefail

echo
echo "$(date '+%Y-%m-%d %H:%M:%S') - Script started"

source /etc/environment

if [ "$(id -u)" -eq 0 ]; then
    echo "Running as root."
else
    echo "Not running as root. Exiting."
    exit 1
fi

cd /home/pi/workspace/pyrfid-jukebox
echo "$(date '+%Y-%m-%d %H:%M:%S') Current directory: $(pwd)"
echo "$(date '+%Y-%m-%d %H:%M:%S') Listing remote branches:"
sudo -u pi git branch -r

echo "$(date '+%Y-%m-%d %H:%M:%S') Fetching from remote..."
sudo -u pi git fetch

branch=${PJ_DEFAULT_BRANCH:-main}

pipinstall=false
echo "$(date '+%Y-%m-%d %H:%M:%S') Checking for updates in requirements.txt..."
if ! sudo -u pi git diff --quiet origin/$branch...HEAD -- requirements.txt; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') New requirements found in requirements.txt"
    pipinstall=true
else
    echo "$(date '+%Y-%m-%d %H:%M:%S') No new requirements in requirements.txt"
fi

repodiffs=false
if [ $(sudo -u pi git rev-parse HEAD) != $(sudo -u pi git rev-parse @{u}) ]; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') New version available. Updating..."
    sudo -u pi git reset --hard origin/$branch
    sudo -u pi git pull
    repodiffs=true
else
    echo "$(date '+%Y-%m-%d %H:%M:%S') No updates found."
fi

echo "$(date '+%Y-%m-%d %H:%M:%S') Checking variables $repodiffs $pipinstall"

if [ "$repodiffs" = true ]; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') Running setup"
    bash setup.sh
    GOCMD=/home/pi/.asdf/shims/go make build
    mv system/soundsprout/soundsprout-server /usr/local/bin/
    sudo supervisorctl restart btconnect
    /usr/local/bin/soundsprout-server templategen --output-dir /tmp
    bash /tmp/runner.sh
fi

if [ "$pipinstall" = true ]; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') Installing requirements"
    sudo -u pi pip3 install -r requirements.txt
fi

if [ "$repodiffs" = true ]; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') Restarting soundsprout_server"
    sudo supervisorctl restart soundsprout_server
fi

echo "$(date '+%Y-%m-%d %H:%M:%S') Sleeping for 5 minutes..."
sleep 5m
