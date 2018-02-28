#!/bin/bash

# Map environment variables to flags for Golang linker's -ldflags usage
function get_linker_ldflags {
    echo -n "-X 'github.com/mysterium/node/cmd.MysteriumAPIURL=${MYSTERIUM_API_URL}' "
    echo -n "-X 'github.com/mysterium/node/cmd/commands/server.natsServerIP=${NATS_SERVER_IP}' "
    [ -n "$TRAVIS_BRANCH" ] && echo -n "-X 'github.com/mysterium/node/version.GitBranch=${TRAVIS_BRANCH}' "
    [ -n "$TRAVIS_COMMIT" ] && echo -n "-X 'github.com/mysterium/node/version.GitCommit=${TRAVIS_COMMIT}' "
    [ -n "$TRAVIS_JOB_NUMBER" ] && echo -n "-X 'github.com/mysterium/node/version.BuildNumber=${TRAVIS_JOB_NUMBER}' "

}
