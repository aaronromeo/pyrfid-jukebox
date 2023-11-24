#!/bin/bash

set -xeuo pipefail

echo
echo "$(date '+%Y-%m-%d %H:%M:%S') - Script started"

touch /home/pi/logs/btconnect.err.log
touch /home/pi/logs/btconnect.out.log
chown pi:pi /home/pi/logs/btconnect.err.log
chown pi:pi /home/pi/logs/btconnect.out.log
echo "

$(date '+%Y-%m-%d %H:%M:%S') - Created btconnect logs." >> /home/pi/logs/btconnect.out.log
echo "

$(date '+%Y-%m-%d %H:%M:%S') - Created btconnect err." >> /home/pi/logs/btconnect.err.log

touch /home/pi/logs/cmus.err.log
touch /home/pi/logs/cmus.out.log
chown pi:pi /home/pi/logs/cmus.err.log
chown pi:pi /home/pi/logs/cmus.out.log
echo "

$(date '+%Y-%m-%d %H:%M:%S') - Created cmus logs." >> /home/pi/logs/cmus.out.log
echo "

$(date '+%Y-%m-%d %H:%M:%S') - Created cmus err." >> /home/pi/logs/cmus.err.log

touch /home/pi/logs/cmus_manager.err.log
touch /home/pi/logs/cmus_manager.out.log
chown pi:pi /home/pi/logs/cmus_manager.err.log
chown pi:pi /home/pi/logs/cmus_manager.out.log
echo "

$(date '+%Y-%m-%d %H:%M:%S') - Created cmus_manager logs." >> /home/pi/logs/cmus_manager.out.log
echo "

$(date '+%Y-%m-%d %H:%M:%S') - Created cmus_manager err." >> /home/pi/logs/cmus_manager.err.log

touch /home/pi/logs/pyrfid-jukebox.err.log
touch /home/pi/logs/pyrfid-jukebox.out.log
chown pi:pi /home/pi/logs/pyrfid-jukebox.err.log
chown pi:pi /home/pi/logs/pyrfid-jukebox.out.log
echo "

$(date '+%Y-%m-%d %H:%M:%S') - Created pyrfid-jukebox logs." >> /home/pi/logs/pyrfid-jukebox.out.log
echo "

$(date '+%Y-%m-%d %H:%M:%S') - Created pyrfid-jukebox err." >> /home/pi/logs/pyrfid-jukebox.err.log

touch /home/pi/logs/gitpull.err.log
touch /home/pi/logs/gitpull.out.log
chown pi:pi /home/pi/logs/gitpull.err.log
chown pi:pi /home/pi/logs/gitpull.out.log
echo "

$(date '+%Y-%m-%d %H:%M:%S') - Created gitpull logs." >> /home/pi/logs/gitpull.out.log
echo "

$(date '+%Y-%m-%d %H:%M:%S') - Created gitpull err." >> /home/pi/logs/gitpull.err.log

sudo supervisorctl start gitpull
sudo supervisorctl start cmus_manager
