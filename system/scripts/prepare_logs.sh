#!/bin/bash

set -euo pipefail

touch /home/pi/btconnect.err.log
touch /home/pi/btconnect.out.log
chown pi:pi /home/pi/btconnect.err.log
chown pi:pi /home/pi/btconnect.out.log

touch /home/pi/cmus.err.log
touch /home/pi/cmus.out.log
chown pi:pi /home/pi/cmus.err.log
chown pi:pi /home/pi/cmus.out.log
