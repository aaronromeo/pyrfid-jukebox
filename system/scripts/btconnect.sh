#!/bin/bash

set -uo pipefail

device="FC:58:FA:8C:E3:A8"

connect_bluetooth() {
    if bluetoothctl connect "$device"; then
        echo "Connected successfully to $device."
        pactl list sinks short
        paplay -p --device=1 /usr/share/sounds/alsa/Front_Center.wav
    else
        echo "Failed to connect."
    fi
}

# Loop indefinitely
while true; do
    # Check if already connected
    current_connection=$(bluetoothctl info "$device" | grep -c "Connected: yes")
    if [ "$current_connection" -eq 0 ]; then
        echo "Attempting to connect to $device..."

        # Start PulseAudio if not running
        pulseaudio --check || pulseaudio --start

        # Ensure Bluetooth is powered on
        powered=$(bluetoothctl show | grep "Powered:" | cut -d ' ' -f 2)
        if [ "$powered" != "yes" ]; then
            bluetoothctl power on
            sleep 10  # Short delay to allow Bluetooth to power on
        fi

        # Try to connect
        connect_bluetooth
    else
        echo "Already connected to $device."
    fi

    # Wait before checking again
    sleep 60
done
