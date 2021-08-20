#!/bin/bash
set -e

export OS_DIR_DATA="/var/lib/mysterium-node"
export OS_DIR_RUN="/var/run/mysterium-node"

# TODO remove this block once all container migrated to new config path.
export OS_DIR_CONFIG="/etc/mysterium-node"
if [ -f $OS_DIR_CONFIG/config.toml ]; then
    mv $OS_DIR_CONFIG/config.toml $OS_DIR_DATA
fi

/usr/local/bin/prepare-run-env.sh

exec /usr/bin/myst \
 --config-dir=$OS_DIR_DATA \
 --script-dir=$OS_DIR_CONFIG \
 --data-dir=$OS_DIR_DATA \
 --runtime-dir=$OS_DIR_RUN \
 --local-service-discovery=false \
 $@
