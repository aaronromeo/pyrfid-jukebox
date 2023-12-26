#!/bin/bash

set -uo pipefail

echo
echo "$(date '+%Y-%m-%d %H:%M:%S') - Script started"

device="88:C6:26:23:95:3F"

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
    else
        status=$(sudo supervisorctl status alsaloop | awk '{print $2}')
        if [[ $status != "RUNNING" && $status != "STARTING" ]]; then
            echo "$(date '+%Y-%m-%d %H:%M:%S') alsaloop is not running or starting. Starting it now..."
            sudo supervisorctl start alsaloop
        else
            echo "$(date '+%Y-%m-%d %H:%M:%S') alsaloop is already running or starting."
        fi

        echo "$(date '+%Y-%m-%d %H:%M:%S') Already connected to $device. Sleeping for 30 seconds."
    fi

    # Wait before checking again
    sleep 30
done
