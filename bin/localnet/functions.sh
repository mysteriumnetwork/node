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

    ${dockerComposeCmd} run local-node init genesis.json
    if [ ! $? -eq 0 ]; then
        print_error "Geth node initialization failed"
        cleanup "$@"
        exit 1
    fi


    ${dockerComposeCmd} up -d broker discovery local-node
    if [ ! $? -eq 0 ]; then
        print_error "Error starting other services"
        cleanup "$@"
        exit 1
    fi

    #deploy MystToken and Payment contracts
    echo "Deploying contracts..."
    ${dockerComposeCmd} run go-runner \
        go run bin/localnet/deployer/*.go \
        --keystore.directory=bin/localnet/account \
        --ether.address=0xa754f0d31411d88e46aed455fa79b9fced122497 \
        --keystore.passphrase `cat bin/localnet/local_acc_password.txt` \
        --geth.url=http://local-node:8545
    if [ ! $? -eq 0 ]; then
        print_error "Error deploying contracts"
        cleanup "$@"
        exit 1
    fi

}

cleanup () {
    setupDockerComposeCmd "$@"

    echo "Cleaning up: $projectName"
    ${dockerComposeCmd} down --remove-orphans --volumes
}