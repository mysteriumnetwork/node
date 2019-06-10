#!/usr/bin/env bash

set -e

NET_ENV=localnet
PROJECT_FILE="bin/${NET_ENV}/docker-compose.yml"

source bin/${NET_ENV}/functions.sh

setup "${NET_ENV}" "bin/${NET_ENV}/publish-ports.yml"
