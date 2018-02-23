#!/bin/bash

set -e

KIND=$1 #client or server
ARCHIVE_NAME=$2 # i.e. output/mysterium_server.tar.gz

ARCHIVE_ROOT=`dirname $ARCHIVE_NAME`
ARCHIVE_DIR="mysterium_${KIND}"
FULL_ARCHIVE_DIR="$ARCHIVE_ROOT/$ARCHIVE_DIR"
mkdir -p "$FULL_ARCHIVE_DIR"

cp "build/$KIND/mysterium_$KIND" "$FULL_ARCHIVE_DIR"
cp -r "bin/${KIND}_package/config" "$FULL_ARCHIVE_DIR"

tar -zcvf "$ARCHIVE_NAME" -C "$ARCHIVE_ROOT" "$ARCHIVE_DIR"
rm -rf "$ARCHIVE_DIR"
