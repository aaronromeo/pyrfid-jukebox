#!/bin/bash

set -euo pipefail

if [ "$(id -u)" -eq 0 ]; then
    echo "Running as root."
else
    echo "Not running as root. Exiting."
    exit 1
fi

cp -R system/scripts ~
cp system/supervisor/conf.d/* /etc/supervisor/conf.d/

mkdir -p /home/pi/logs

chown -R pi:pi /home/pi/logs

supervisorctl reread
supervisorctl update
