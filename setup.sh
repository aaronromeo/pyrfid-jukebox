#!/bin/bash

set -euo pipefail

if [ "$(id -u)" -eq 0 ]; then
    echo "Running as root."
else
    echo "Not running as root. Exiting."
    exit 1
fi

if ! git config --global --get-all safe.directory 2>/dev/null | grep -qxF /home/pi/workspace/pyrfid-jukebox; then
    git config --global --add safe.directory /home/pi/workspace/pyrfid-jukebox
fi

cp -R system/scripts ~
rm /etc/supervisor/conf.d/*
cp system/supervisor/conf.d/* /etc/supervisor/conf.d/

mkdir -p /home/pi/logs

chown -R pi:pi /home/pi/logs

mkdir -p /etc/sudoers.d
cp system/sudoers.d/soundsprout-portal /etc/sudoers.d/soundsprout-portal
chmod 440 /etc/sudoers.d/soundsprout-portal
visudo -cf /etc/sudoers.d/soundsprout-portal

supervisorctl reread
supervisorctl update
