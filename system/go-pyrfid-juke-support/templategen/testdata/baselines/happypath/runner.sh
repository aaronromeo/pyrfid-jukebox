#!/bin/bash

mkdir -p /home/pi/.config/cmus
sudo mv <TEMPDIR>/config-cmus-autosave.txt /home/pi/.config/cmus/autosave
sudo chown pi /home/pi/.config/cmus/autosave
sudo supervisorctl restart cmus_manager
mkdir -p /home/pi
sudo mv <TEMPDIR>/asoundrc.txt /home/pi/.asoundrc
sudo chown pi /home/pi/.asoundrc
sudo supervisorctl restart btconnect
sudo alsactl restore
