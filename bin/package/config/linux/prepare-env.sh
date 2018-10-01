#!/usr/bin/env bash

if [ ! -d /dev/net ]; then
    mkdir -p /dev/net
fi

if [ ! -c /dev/net/tun ]; then
    mknod /dev/net/tun c 10 200
fi

chmod 0666 /dev/net/tun

