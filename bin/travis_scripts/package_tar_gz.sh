#!/bin/bash

set -e

KIND=$1 #client or server
ARCHIVE_NAME=$2 # i.e. output/mysterium_server.tar.gz

ARCHIVE_DIR=`dirname $ARCHIVE_NAME`/mysterium_${KIND}
mkdir -p $ARCHIVE_DIR

cp "build/$KIND/mysterium_$KIND" "$ARCHIVE_DIR"
cp -r "bin/${KIND}_package/config" "$ARCHIVE_DIR"

tar -zcvf $ARCHIVE_NAME $ARCHIVE_DIR && rm -rf $ARCHIVE_DIR