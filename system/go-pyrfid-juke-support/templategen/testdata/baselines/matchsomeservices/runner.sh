#!/bin/bash

mv asoundrc.txt ./testdata/destination/matchsomeservices/pi/.asoundrc
chown pi ./testdata/destination/matchsomeservices/pi/.asoundrc
sudo supervisorctl btconnect reload
sudo supervisorctl btconnect restart
