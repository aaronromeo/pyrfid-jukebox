#!/bin/bash

set -uo pipefail

echo
echo "$(date '+%Y-%m-%d %H:%M:%S') - Script started"

device="FC:58:FA:8C:E3:A8"

connect_bluetooth() {
    if bluetoothctl connect "$device"; then
        echo "Connected successfully to $device."
        aplay -D bluealsa /usr/share/sounds/alsa/Front_Center.wav
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
        echo "$(date '+%Y-%m-%d %H:%M:%S') Already connected to $device. Sleeping for 1 minute."
    fi

    # Wait before checking again
    sleep 60
done
