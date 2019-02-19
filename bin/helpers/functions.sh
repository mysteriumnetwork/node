#!/bin/bash

# Map environment variables to flags for Golang linker's -ldflags usage
function get_linker_ldflags {
    [ -n "$BUILD_BRANCH" ] && echo -n "-X 'github.com/mysteriumnetwork/node/metadata.BuildBranch=${BUILD_BRANCH}' "
    [ -n "$BUILD_COMMIT" ] && echo -n "-X 'github.com/mysteriumnetwork/node/metadata.BuildCommit=${BUILD_COMMIT}' "
    [ -n "$BUILD_NUMBER" ] && echo -n "-X 'github.com/mysteriumnetwork/node/metadata.BuildNumber=${BUILD_NUMBER}' "
    [ -n "$BUILD_VERSION" ] && echo -n "-X 'github.com/mysteriumnetwork/node/metadata.Version=${BUILD_VERSION}' "
}

function copy_config {
    local OS_DIR=$1
    local DST_DIR=$2
    cp -vrp "bin/package/config/common/" ${DST_DIR}/config
    if [[ -d "bin/package/config/${OS_DIR}/" ]]; then
        cp -vrp "bin/package/config/${OS_DIR}/." ${DST_DIR}/config
    fi
}
