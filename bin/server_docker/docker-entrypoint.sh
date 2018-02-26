#!/bin/bash
set -e

OS_DIR_CONFIG="/etc/mysterium-node"
OS_DIR_DATA="/var/lib/mysterium-node"
OS_DIR_RUN="/var/run/mysterium-node"

if [ ! -d /dev/net ]; then
    mkdir -p /dev/net
fi
if [ ! -c /dev/net/tun ]; then
    mknod /dev/net/tun c 10 200
fi

if [ ! -d "$OS_DIR_RUN" ]; then
    mkdir -p $OS_DIR_RUN
fi

exec /usr/bin/mysterium_server \
    --config-dir=$OS_DIR_CONFIG \
    --data-dir=$OS_DIR_DATA \
    --runtime-dir=$OS_DIR_RUN \
    --identity=$MYSTERIUM_SERVER_IDENTITY \
    --identity.passphrase=$MYSTERIUM_SERVER_IDENTITY_PASSPHRASE \
    --location.database=$MYSTERIUM_SERVER_COUNTRY_DATABASE \
    --location.country=$MYSTERIUM_SERVER_COUNTRY \
    --discovery-address=$MYSTERIUM_DISCOVERY_ADDRESS \
    --broker-address=$MYSTERIUM_BROKER_ADDRESS
