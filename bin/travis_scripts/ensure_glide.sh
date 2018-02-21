#!/bin/bash

TOOLS_PATH=$1
GLIDE_VERSION=$2

glide --version
if [ $? -eq 0 ]; then
    echo "Glide found"
else
    wget "https://github.com/Masterminds/glide/releases/download/$GLIDE_VERSION/glide-$GLIDE_VERSION-linux-amd64.tar.gz"
    tar -vxz -C $TOOLS_PATH --strip=1 -f glide-$GLIDE_VERSION-linux-amd64.tar.gz
fi
