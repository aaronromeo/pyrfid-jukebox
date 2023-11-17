#!/bin/bash

set -euo pipefail

if [ "$(id -u)" -eq 0 ]; then
    echo "Running as root."
else
    echo "Not running as root. Exiting."
    exit 1
fi

cp system/supervisor/conf.d/* /etc/supervisor/conf.d/

mkdir /home/pi/logs
mkdir /home/pi/scripts

cp -v system/scripts/btconnect.sh /home/pi/scripts
cp -v system/scripts/prepare_logs.sh /home/pi/scripts

chown -R pi:pi /home/pi/logs
chown -R pi:pi /home/pi/scripts

supervisorctl reread
supervisorctl update
