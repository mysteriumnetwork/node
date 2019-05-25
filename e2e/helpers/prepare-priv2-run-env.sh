#!/usr/bin/env bash

ip route add 172.30.0.0/24 via 10.100.1.2

if [ ! -d "$OS_DIR_RUN" ]; then
    mkdir -p $OS_DIR_RUN
fi

if [ ! -d "$OS_DIR_DATA" ]; then
    mkdir -p $OS_DIR_DATA
fi
