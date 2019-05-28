#!/usr/bin/env bash

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

