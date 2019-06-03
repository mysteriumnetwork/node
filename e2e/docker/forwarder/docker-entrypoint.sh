#!/bin/bash
set -e

setDefaultRoute() {
    GW=$1;
    ip route del default
    ip route add default via ${GW}
}

if [ -n "$EXT_NAT" ]; then
    echo "external NAT for containers via: ${EXT_NAT}"
    iptables -t nat -A POSTROUTING -o `ip r get ${EXT_NAT} | awk '{ print $3 }'` ! -d 172.16.0.0/12 -j MASQUERADE
fi

if [ -n "$GATEWAY" ]; then
    echo "new gateway: ${GATEWAY}"
    iptables -t nat -A POSTROUTING -o `ip r get ${GATEWAY} | awk '{ print $3 }'` -j MASQUERADE
    setDefaultRoute ${GATEWAY}
fi

if [ -n "$DEFAULT_ROUTE" ]; then
    echo "new default route: ${DEFAULT_ROUTE}"
    setDefaultRoute ${DEFAULT_ROUTE}
fi

named -c /etc/bind/named.conf -g -u named
