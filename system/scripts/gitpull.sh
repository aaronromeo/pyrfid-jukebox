#!/bin/bash

set -xeuo pipefail


cd /home/pi/workspace/pyrfid-jukebox
sudo -u pi git fetch

if [ $(git rev-parse HEAD) != $(git rev-parse @{u}) ]; then
    echo "New version available. Updating..."
    sudo -u pi git pull

    echo "Installing requirements"
    sudo -u pi pip3 install -r requirements.txt

    echo "Running setup"
    bash setup.sh

    echo "Restarting pyrfid_jukebox"
    supervisord restart "pyrfid_jukebox"
else
    echo "No updates found."
fi
