#!/bin/bash

set -xeuo pipefail

pushd /home/pi/workspace/pyrfid-jukebox
sudo -u pi git reset --hard origin/main
sudo bash setup.sh
popd

echo
echo "$(date '+%Y-%m-%d %H:%M:%S') - Script started"

touch /home/pi/logs/btconnect.err.log
touch /home/pi/logs/btconnect.out.log
chown pi:pi /home/pi/logs/btconnect.err.log
chown pi:pi /home/pi/logs/btconnect.out.log
echo "Created btconnect logs." >> /home/pi/logs/btconnect.out.log
echo "Created btconnect err." >> /home/pi/logs/btconnect.err.log

touch /home/pi/logs/cmus.err.log
touch /home/pi/logs/cmus.out.log
chown pi:pi /home/pi/logs/cmus.err.log
chown pi:pi /home/pi/logs/cmus.out.log
echo "Created cmus logs." >> /home/pi/logs/cmus.out.log
echo "Created cmus err." >> /home/pi/logs/cmus.err.log

touch /home/pi/logs/pyrfid-jukebox.err.log
touch /home/pi/logs/pyrfid-jukebox.out.log
chown pi:pi /home/pi/logs/pyrfid-jukebox.err.log
chown pi:pi /home/pi/logs/pyrfid-jukebox.out.log
echo "Created pyrfid-jukebox logs." >> /home/pi/logs/pyrfid-jukebox.out.log
echo "Created pyrfid-jukebox err." >> /home/pi/logs/pyrfid-jukebox.err.log

touch /home/pi/logs/gitpull.err.log
touch /home/pi/logs/gitpull.out.log
chown pi:pi /home/pi/logs/gitpull.err.log
chown pi:pi /home/pi/logs/gitpull.out.log
echo "Created gitpull logs." >> /home/pi/logs/gitpull.out.log
echo "Created gitpull err." >> /home/pi/logs/gitpull.err.log
