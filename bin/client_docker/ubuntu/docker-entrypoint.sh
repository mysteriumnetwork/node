#!/bin/bash
set -e

OS_DIR_CONFIG="/etc/mysterium-client"
OS_DIR_DATA="/var/lib/mysterium-client"
OS_DIR_RUN="/var/run/mysterium-client"

if [ ! -d "$OS_DIR_RUN" ]; then
    mkdir -p $OS_DIR_RUN
fi

exec /usr/bin/mysterium_client \
    --config-dir=$OS_DIR_CONFIG \
    --data-dir=$OS_DIR_DATA \
    --runtime-dir=$OS_DIR_RUN \
    --tequilapi.address=0.0.0.0 \
    --tequilapi.port=$MYSTERIUM_CLIENT_TEQUILAPI_PORT \
    --discovery-address=$MYSTERIUM_DISCOVERY_ADDRESS \
    $@
