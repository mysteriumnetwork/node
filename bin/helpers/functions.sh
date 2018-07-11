#!/bin/bash

# Map environment variables to flags for Golang linker's -ldflags usage
function get_linker_ldflags {
    [ -n "$BUILD_BRANCH" ] && echo -n "-X 'github.com/mysterium/node/metadata.BuildBranch=${BUILD_BRANCH}' "
    [ -n "$BUILD_COMMIT" ] && echo -n "-X 'github.com/mysterium/node/metadata.BuildCommit=${BUILD_COMMIT}' "
    [ -n "$BUILD_NUMBER" ] && echo -n "-X 'github.com/mysterium/node/metadata.BuildNumber=${BUILD_NUMBER}' "
    [ -n "$BUILD_VERSION" ] && echo -n "-X 'github.com/mysterium/node/metadata.Version=${BUILD_VERSION}' "
}
