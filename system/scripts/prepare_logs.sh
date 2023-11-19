#!/bin/bash

set -xeuo pipefail

touch /home/pi/logs/btconnect.err.log
touch /home/pi/logs/btconnect.out.log
chown pi:pi /home/pi/logs/btconnect.err.log
chown pi:pi /home/pi/logs/btconnect.out.log

touch /home/pi/logs/cmus.err.log
touch /home/pi/logs/cmus.out.log
chown pi:pi /home/pi/logs/cmus.err.log
chown pi:pi /home/pi/logs/cmus.out.log

touch /home/pi/logs/pyrfid-jukebox.err.log
touch /home/pi/logs/pyrfid-jukebox.out.log
chown pi:pi /home/pi/logs/pyrfid-jukebox.err.log
chown pi:pi /home/pi/logs/pyrfid-jukebox.out.log

touch /home/pi/logs/gitpull.err.log
touch /home/pi/logs/gitpull.out.log
chown pi:pi /home/pi/logs/gitpull.err.log
chown pi:pi /home/pi/logs/gitpull.out.log
