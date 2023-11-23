#!/bin/bash

set -euo pipefail

if [ "$(id -u)" -eq 0 ]; then
    echo "Running as root."
else
    echo "Not running as root. Exiting."
    exit 1
fi

cp system/supervisor/conf.d/* /etc/supervisor/conf.d/

mkdir -p /home/pi/logs
mkdir -p /home/pi/scripts

cp -v system/scripts/*.sh /home/pi/scripts
rm /home/pi/scripts/gitpull.sh
chmod +x /home/pi/scripts/*.sh

chown -R pi:pi /home/pi/logs
chown -R pi:pi /home/pi/scripts

supervisorctl reread
supervisorctl update
