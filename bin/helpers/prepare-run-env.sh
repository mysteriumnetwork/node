#!/usr/bin/env bash

setDefaultRoute() {
    GW=$1;
    ip route del default
    ip route add default via ${GW}
}

if [ -n "$GATEWAY" ]; then
    echo "new gateway: ${GATEWAY}"
    iptables -t nat -A POSTROUTING -o `ip r get ${GATEWAY} | awk '{ print $3 }'` -j MASQUERADE
    setDefaultRoute ${GATEWAY}
fi

if [ -n "$DEFAULT_ROUTE" ]; then
    echo "new default route: ${DEFAULT_ROUTE}"
    setDefaultRoute ${DEFAULT_ROUTE}
fi

if [ -n "$PUBLIC_ROUTE" ]; then
    echo "adding route: ${PUBLIC_ROUTE}"
    eval ${PUBLIC_ROUTE}
fi

if [ ! -d "$OS_DIR_RUN" ]; then
    mkdir -p $OS_DIR_RUN
fi

if [ ! -d "$OS_DIR_DATA" ]; then
    mkdir -p $OS_DIR_DATA
fi

