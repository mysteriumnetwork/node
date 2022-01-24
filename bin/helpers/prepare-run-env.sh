#!/usr/bin/env bash

function setDefaultRoute {
    GW=$1;
    ip route del default
    ip route add default via ${GW}
}

function ensure_paths {
    iptables_path=`which iptables`
    if [[ ${iptables_path} == "" ]]; then
      echo "required dependency missing: iptables"
      exit 1
    fi

    # validate utility against valid system paths
    basepath=${iptables_path%/*}
    echo "iptables basepath detected: ${basepath}"
    if ! [[ ${basepath} =~ (^/usr/sbin|^/sbin|^/bin|^/usr/bin) ]]; then
      echo "invalid basepath for dependency - check if system PATH has not been altered"
      exit 1
    fi

    iptables_required_path="/usr/sbin/iptables"

    if ! env [ -x "${iptables_required_path}" ]; then
        ln -s ${iptables_path} ${iptables_required_path}
    fi
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

ensure_paths
