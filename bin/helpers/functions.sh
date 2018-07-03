#!/bin/bash

# Map environment variables to flags for Golang linker's -ldflags usage
function get_linker_ldflags {
    [ -n "$BRANCH" ] && echo -n "-X 'github.com/mysterium/node/metadata.BuildBranch=${BRANCH}' "
    [ -n "$COMMIT" ] && echo -n "-X 'github.com/mysterium/node/metadata.BuildCommit=${COMMIT}' "
    [ -n "$BUILD_NUMBER" ] && echo -n "-X 'github.com/mysterium/node/metadata.BuildNumber=${BUILD_NUMBER}' "

}
