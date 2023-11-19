#!/bin/bash
if ! screen -list | grep -q "cmus"; then
  /usr/bin/screen -dmS cmus /usr/bin/cmus
fi
