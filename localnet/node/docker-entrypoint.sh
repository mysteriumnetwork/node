#!/bin/bash
set -e

export OS_DIR_CONFIG="/etc/mysterium-node"
export OS_DIR_DATA="/var/lib/mysterium-node"
export OS_DIR_RUN="/var/run/mysterium-node"

/usr/local/bin/prepare-run-env.sh

while true
do
    echo "Press [CTRL+C] to stop.."
    sleep 10
done
