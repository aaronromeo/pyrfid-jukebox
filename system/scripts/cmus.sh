#!/bin/bash

set -xeuo pipefail

if ! screen -list | grep -q "cmus"; then
  /usr/bin/screen -dmS cmus /usr/bin/cmus
fi
