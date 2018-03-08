#!/bin/bash

# Map environment variables to flags for Golang linker's -ldflags usage
function get_linker_ldflags {
    echo -n "-X 'github.com/mysterium/node/cmd.MysteriumAPIURL=${MYSTERIUM_API_URL}' "
    echo -n "-X 'github.com/mysterium/node/cmd/commands/server.natsServerIP=${NATS_SERVER_IP}' "
    [ -n "$BRANCH" ] && echo -n "-X 'github.com/mysterium/node/version.gitBranch=${BRANCH}' "
    [ -n "$COMMIT" ] && echo -n "-X 'github.com/mysterium/node/version.gitCommit=${COMMIT}' "
    [ -n "$BUILD_NUMBER" ] && echo -n "-X 'github.com/mysterium/node/version.buildNumber=${BUILD_NUMBER}' "

}
