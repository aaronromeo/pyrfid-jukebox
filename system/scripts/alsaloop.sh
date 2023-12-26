#!/bin/bash

if lsmod | grep -q snd_aloop; then
    echo "snd_aloop module is loaded."
else
    echo "snd_aloop module is not loaded."
    sudo modprobe snd-aloop
fi

alsaloop -C hw:Loopback,1,0 -P bluealsa:DEV=88:C6:26:23:95:3F,PROFILE=a2dp -c 2 -r 48000 -f s16_le -t 200000
