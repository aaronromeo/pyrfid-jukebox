#!/bin/bash

set -uo pipefail

echo
echo "$(date '+%Y-%m-%d %H:%M:%S') - Script started"

device="AC:BF:71:DA:D2:55"

connect_bluetooth() {
    sudo modprobe snd-aloop
    if bluetoothctl connect "$device"; then
        echo "Connected successfully to $device."

        bluealsa-aplay -l
    else
        echo "Failed to connect."
    fi
}

# Loop indefinitely
while true; do

    if diff /home/pi/.asoundrc /home/pi/workspace/pyrfid-jukebox/system/home/.asoundrc >/dev/null; then
        echo "$(date '+%Y-%m-%d %H:%M:%S') .asoundrc has changed in repo. Copying over system config..."
        cp /home/pi/workspace/pyrfid-jukebox/system/home/.asoundrc /home/pi/.asoundrc
        sudo alsactl restore
    fi

    # Check if already connected
    current_connection=$(bluetoothctl info "$device" | grep -c "Connected: yes")
    current_alsa_status=$(sudo service bluealsa status | grep -c "active (running)")
    if [ "$current_connection" -eq 0 ]; then
        echo "$(date '+%Y-%m-%d %H:%M:%S') Attempting to connect to $device..."

        # Start BlueALSA if not running
        if [ "$current_alsa_status" -eq 0 ]; then
            sudo service bluealsa start
        fi

        # Ensure Bluetooth is powered on
        powered=$(bluetoothctl show | grep "Powered:" | cut -d ' ' -f 2)
        if [ "$powered" != "yes" ]; then
            bluetoothctl power on
            sleep 10  # Short delay to allow Bluetooth to power on
        fi

        # Try to connect
        connect_bluetooth
    fi

    # Wait before checking again
    sleep 30
done
