#!/bin/bash

mkdir -p /home/pi/.config/cmus
mv config-cmus-autosave.txt /home/pi/.config/cmus/autosave
chown pi /home/pi/.config/cmus/autosave
sudo supervisorctl cmus_manager reload
sudo supervisorctl cmus_manager restart
mkdir -p /home/pi
mv asoundrc.txt /home/pi/.asoundrc
chown pi /home/pi/.asoundrc
sudo supervisorctl btconnect reload
sudo supervisorctl btconnect restart
