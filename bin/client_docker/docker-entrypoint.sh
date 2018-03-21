#!/bin/bash
set -e

export OS_DIR_CONFIG="/etc/mysterium-client"
export OS_DIR_DATA="/var/lib/mysterium-client"
export OS_DIR_RUN="/var/run/mysterium-client"

/usr/local/bin/prepare-run-env.sh

exec /usr/bin/mysterium_client \
 --config-dir=$OS_DIR_CONFIG \
 --data-dir=$OS_DIR_DATA \
 --runtime-dir=$OS_DIR_RUN \
 --tequilapi.address=0.0.0.0 \
 $@
