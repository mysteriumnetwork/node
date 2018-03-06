#!/usr/bin/env bash

set -e

PROJECT_NAME="nodetest"
SUCCESS_COLOR='\033[0;32m' # green
FAILURE_COLOR="\033[0;31m" # red
DEFAULT_COLOR="\033[0;0m"

print_success () {
    echo -e $SUCCESS_COLOR$1$DEFAULT_COLOR
}

print_error () {
    echo -e $FAILURE_COLOR$1$DEFAULT_COLOR
}

setup () {
    sudo docker-compose -p $PROJECT_NAME build
    if [ ! $? -eq 0 ]; then
        print_error "Building docker images failed"
        exit 1
    fi

    docker-compose -p $PROJECT_NAME up -d
    if [ ! $? -eq 0 ]; then
        print_error "Starting built docker images failed"
        exit 1
    fi
}

cleanup () {
    echo "Cleaning up"
    docker-compose -p $PROJECT_NAME down
}


setup

result=`curl localhost:4050/connection`
if [ "$result" != '{"status":"NotConnected"}' ]
then
    print_error "Unexpected status response: $result"
    cleanup
    exit 1
fi

print_success "Tests passed"
cleanup
exit 0
