#!/bin/bash

set -euxo pipefail

device="FC:58:FA:8C:E3:A8"
max_attempts=5
attempt=1
connected=0

connect_bluetooth() {
    if bluetoothctl connect "$device"; then
        echo "Connected successfully to $device."
        return 0
    else
        echo "Failed to connect."
        return 1
    fi
}

# Check if already connected
current_connection=$(bluetoothctl info "$device" | grep -c "Connected: yes")
if [ "$current_connection" -eq 1 ]; then
    echo "Already connected to $device."
    exit 0
fi

# Start PulseAudio if not running
if ! pulseaudio --check; then
    echo "Starting PulseAudio..."
    pulseaudio --start
else
    echo "PulseAudio is already running."
fi

# Start bluetooth service if not running
powered=$(bluetoothctl show | grep "Powered:" | cut -d ' ' -f 2)
if [ "$powered" = "yes" ]; then
    echo "Bluetooth is powered on."
else
    echo "Bluetooth is not powered on."
    bluetoothctl power on
    echo "Sleeping for 120 seconds while bluetooth powers on..."
    sleep 120
fi

echo "Attempt to connect to $device..."

if connect_bluetooth; then
    pactl list sinks short
    paplay -p --device=1 /usr/share/sounds/alsa/Front_Center.wav
else
    exit 1
fi
