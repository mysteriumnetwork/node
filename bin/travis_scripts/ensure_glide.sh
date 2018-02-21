#!/bin/bash

TOOLS_PATH=$1
GLIDE_VERSION=$2


OS_ARCH="linux-amd64"
if [ "$TRAVIS_OS_NAME"=="osx" ]; then
    OS_ARCH="darwin-amd64"
fi

glide --version
if [ $? -eq 0 ]; then
    echo "Glide found"
else
    wget "https://github.com/Masterminds/glide/releases/download/$GLIDE_VERSION/glide-$GLIDE_VERSION-$OS_ARCH.tar.gz"
    tar -vxz -C $TOOLS_PATH --strip=1 -f glide-$GLIDE_VERSION-$OS_ARCH.tar.gz
fi
