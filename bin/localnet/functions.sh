#!/usr/bin/env bash

source bin/helpers/output.sh


PROJECT_FILE="bin/localnet/docker-compose.yml"

setupDockerComposeCmd() {
    projectName=$1; shift;

    projectFiles=("-f ${PROJECT_FILE}")

    for extensionFile in "$@"; do
        projectFiles=("${projectFiles[@]}" "-f ${extensionFile}")
    done

    dockerComposeCmd="docker-compose ${projectFiles[@]} -p $projectName"
}

setup () {

    setupDockerComposeCmd "$@"
    echo "Setting up: $projectName"

    ${dockerComposeCmd} up -d db # start database first - it takes about 10 sec untils db startsup, and otherwise db migration fails
    if [ ! $? -eq 0 ]; then
        print_error "Db startup failed"
        cleanup "$@"
        exit 1
    fi

    echo "Waiting for db to become up"
    while ! ${dockerComposeCmd} exec db mysqladmin ping --protocol=TCP --silent; do
        echo -n "."
        sleep 1
    done
    sleep 2 #even after successful TCP connection we still hit db not ready yet sometimes
    echo "Database is up"

    ${dockerComposeCmd} run --entrypoint bin/db-upgrade discovery
    if [ ! $? -eq 0 ]; then
        print_error "Db migration failed"
        cleanup "$@"
        exit 1
    fi

    ${dockerComposeCmd} up -d broker discovery
    if [ ! $? -eq 0 ]; then
        print_error "Starting built docker images failed"
        cleanup "$@"
        exit 1
    fi
}

cleanup () {
    setupDockerComposeCmd "$@"

    echo "Cleaning up: $projectName"
    ${dockerComposeCmd} down --remove-orphans
}