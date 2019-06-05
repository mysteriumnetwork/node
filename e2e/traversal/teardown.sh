#!/usr/bin/env bash

set -e

NET_ENV=traversal
PROJECT_FILE="e2e/${NET_ENV}/docker-compose.yml"

source bin/localnet/functions.sh

cleanup "${NET_ENV}"
