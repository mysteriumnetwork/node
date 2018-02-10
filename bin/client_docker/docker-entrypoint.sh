#!/bin/bash
set -e

OS_DIR_CONFIG="/etc/mysterium-client"
OS_DIR_DATA="/var/lib/mysterium-client"
OS_DIR_RUN="/var/run/mysterium-client"

exec /usr/bin/mysterium_client \
    --config-dir=$OS_DIR_CONFIG \
    --data-dir=$OS_DIR_DATA \
    --runtime-dir=$OS_DIR_RUN