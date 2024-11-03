#!/bin/bash
set -euo pipefail

sudo mv <TEMPDIR>/asoundrc.txt ./testdata/destination/matchsomeservices/pi/.asoundrc
sudo chown pi ./testdata/destination/matchsomeservices/pi/.asoundrc
sudo supervisorctl restart btconnect
sudo alsactl restore
