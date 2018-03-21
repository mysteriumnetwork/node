#!/usr/bin/env bash

if [ ! -d /dev/net ]; then
    mkdir -p /dev/net
fi
if [ ! -c /dev/net/tun ]; then
    mknod /dev/net/tun c 10 200
fi

if [ ! -d "$OS_DIR_RUN" ]; then
    mkdir -p $OS_DIR_RUN
fi

if [ ! -d "$OS_DIR_DATA" ]; then
    mkdir -p $OS_DIR_DATA
fi

