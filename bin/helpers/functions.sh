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

ANDROID_NDK_VERSION="27.1.12297006"

function gomobile_bind_android {
    local OUT=$1
    local TAGS=$2
    local SDK="${ANDROID_HOME:-$ANDROID_SDK_ROOT}"
    if [[ -z "$SDK" ]]; then
        print_error "ANDROID_HOME is not set"
        return 1
    fi
    export ANDROID_HOME="$SDK"
    export ANDROID_NDK_HOME="${SDK}/ndk/${ANDROID_NDK_VERSION}"
    if [[ ! -d "$ANDROID_NDK_HOME" ]]; then
        print_error "NDK ${ANDROID_NDK_VERSION} missing, install: sdkmanager \"ndk;${ANDROID_NDK_VERSION}\""
        return 1
    fi
    mkdir -p "$(dirname "$OUT")"
    local TOOLS_BIN="$PWD/build/gomobile-bin"
    GOBIN="$TOOLS_BIN" go install golang.org/x/mobile/cmd/gomobile golang.org/x/mobile/cmd/gobind
    PATH="$TOOLS_BIN:$PATH" gomobile bind \
        -target=android/arm64,android/arm,android/amd64,android/386 \
        -androidapi 21 \
        -tags "$TAGS" \
        -ldflags "-w -s $(get_linker_ldflags) -checklinkname=0 -extldflags=-Wl,-z,max-page-size=16384" \
        -o "$OUT" \
        ./mobile/mysterium
}
