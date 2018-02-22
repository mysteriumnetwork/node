#!/bin/bash

# Map environment variables to flags for Golang linker's -ldflags usage
function get_linker_ldflags {
    echo "
        -X 'github.com/mysterium/node/cmd/commands/server.MysteriumApiUrl=${MYSTERIUM_API_URL}'
        -X 'github.com/mysterium/node/cmd/commands/server.natsServerIP=${NATS_SERVER_IP}'
    "
}
