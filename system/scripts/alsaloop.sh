#!/bin/bash

if lsmod | grep -q snd_aloop; then
    echo "snd_aloop module is loaded."
else
    echo "snd_aloop module is not loaded."
    sudo modprobe snd-aloop
fi

alsaloop -C hw:Loopback,1,0 -P bluealsa:DEV=FC:58:FA:8C:E3:A8,PROFILE=a2dp -c 2 -r 48000 -f s16_le -t 200000
