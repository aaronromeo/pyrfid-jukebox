#!/bin/bash

set -euo pipefail

echo
echo "$(date '+%Y-%m-%d %H:%M:%S') - Script started"

if [ -f /home/pi/.soundsprout/conf ]; then
    set -a
    source /home/pi/.soundsprout/conf
    set +a
fi

if [ "$(id -u)" -eq 0 ]; then
    echo "Running as root."
else
    echo "Not running as root. Exiting."
    exit 1
fi

# Deploy state. last_deploy records the commit that was last successfully
# deployed (binary installed AND services restarted); anything else - a new
# HEAD, the daily force, or the force flag - triggers a deploy attempt.
STATE_DIR=/home/pi/.soundsprout
LAST_DEPLOY="$STATE_DIR/last_deploy"
LAST_FORCE="$STATE_DIR/last_force"
FORCE_FLAG="$STATE_DIR/force_rebuild"

cd /home/pi/workspace/pyrfid-jukebox
echo "$(date '+%Y-%m-%d %H:%M:%S') Current directory: $(pwd)"
echo "$(date '+%Y-%m-%d %H:%M:%S') Listing remote branches:"
sudo -u pi git branch -r

echo "$(date '+%Y-%m-%d %H:%M:%S') Fetching from remote..."
if ! sudo -u pi git fetch; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') Fetch failed (likely a transient network or DNS issue). Retrying in 5 minutes."
    sleep 5m
    exit 0
fi

branch=${PJ_DEFAULT_BRANCH:-main}

if [ "$(sudo -u pi git rev-parse HEAD)" != "$(sudo -u pi git rev-parse @{u})" ]; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') New version available. Updating..."
    sudo -u pi git reset --hard origin/$branch
    sudo -u pi git pull
else
    echo "$(date '+%Y-%m-%d %H:%M:%S') No updates found."
fi

head_commit=$(sudo -u pi git rev-parse HEAD)
deployed=$(cat "$LAST_DEPLOY" 2>/dev/null || echo none)

repodiffs=false
if [ "$head_commit" != "$deployed" ]; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') Deploy due: HEAD $head_commit (last deployed: $deployed)."
    repodiffs=true
fi

# Force one rebuild per day (first cycle after midnight), or any time the
# flag file exists: sudo touch /home/pi/.soundsprout/force_rebuild
today=$(date +%F)
force=false
if [ -f "$FORCE_FLAG" ]; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') Forced rebuild requested via $FORCE_FLAG."
    force=true
elif [ "$(cat "$LAST_FORCE" 2>/dev/null || echo none)" != "$today" ]; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') Daily forced rebuild."
    force=true
fi

pipinstall=false
if [ "$deployed" = none ]; then
    pipinstall=true
elif ! sudo -u pi git diff --quiet "$deployed" HEAD -- requirements.txt; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') requirements.txt changed since last deploy."
    pipinstall=true
fi

# Every step must fail the whole deploy (|| return 1); relying on set -e
# here is unreliable because the function is called from an `if` context.
# GOFLAGS=-buildvcs=false: go build's VCS stamping runs git as root inside
# the pi-owned repo and fails with "dubious ownership"; the deployed binary
# does not use VCS stamping.
deploy() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') Running setup"
    bash setup.sh || return 1
    GOFLAGS=-buildvcs=false GOCMD=/home/pi/.asdf/shims/go make build || return 1
    mv system/soundsprout/soundsprout-server /usr/local/bin/ || return 1
    sudo supervisorctl restart btconnect || return 1
    sudo supervisorctl restart portal || return 1
    rm -f /tmp/runner.sh
    /usr/local/bin/soundsprout-server templategen --output-dir /tmp || return 1
    bash /tmp/runner.sh || return 1
    if [ "$pipinstall" = true ]; then
        echo "$(date '+%Y-%m-%d %H:%M:%S') Installing requirements"
        sudo -u pi /home/pi/workspace/pyrfid-jukebox/env/bin/pip install -r requirements.txt || return 1
    fi
    sudo supervisorctl restart soundsprout_server || return 1
}

if [ "$repodiffs" = true ] || [ "$force" = true ]; then
    echo "$today" > "$LAST_FORCE"
    rm -f "$FORCE_FLAG"
    if deploy; then
        echo "$head_commit" > "$LAST_DEPLOY"
        echo "$(date '+%Y-%m-%d %H:%M:%S') Deploy of $head_commit complete."
    else
        echo "$(date '+%Y-%m-%d %H:%M:%S') Deploy FAILED; will retry next cycle." >&2
    fi
fi

echo "$(date '+%Y-%m-%d %H:%M:%S') Sleeping for 5 minutes..."
sleep 5m
