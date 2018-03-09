#!/bin/bash

TOOLS_PATH=$1
DEP_VERSION=$2


OS_ARCH="linux-amd64"
if [ "$TRAVIS_OS_NAME"=="osx" ]; then
    OS_ARCH="darwin-amd64"
fi

dep version
if [ $? -eq 0 ]; then
    echo "Dep found"
else
    # TODO: fetch frozen dep version
    go get -u github.com/golang/dep/cmd/dep
    echo "dep installed:"
    dep version
#    wget "https://github.com/Masterminds/glide/releases/download/$GLIDE_VERSION/glide-$GLIDE_VERSION-$OS_ARCH.tar.gz"
#    tar -vxz -C $TOOLS_PATH --strip=1 -f glide-$GLIDE_VERSION-$OS_ARCH.tar.gz
fi
