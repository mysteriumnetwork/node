#!/bin/bash

set -e

ARCHIVE_NAME=$1 # i.e. output/mysterium_server.tar.gz

ARCHIVE_ROOT=`dirname $ARCHIVE_NAME`
ARCHIVE_DIR="mysterium_server"
FULL_ARCHIVE_DIR="$ARCHIVE_ROOT/$ARCHIVE_DIR"
mkdir -p "$FULL_ARCHIVE_DIR"

cp "build/server/mysterium_server" "$FULL_ARCHIVE_DIR"
cp -r "bin/server_package/config" "$FULL_ARCHIVE_DIR"

tar -zcvf "$ARCHIVE_NAME" -C "$ARCHIVE_ROOT" "$ARCHIVE_DIR"
rm -rf "$FULL_ARCHIVE_DIR"
