#!/bin/sh

# Install latest release of myst for debian/raspbian
#
# Variables:
# - SNAPSHOT (default: false) - set to "true" to install development snapshot
#

set -e

if [[ "$SNAPSHOT" == "true" ]]; then
    releases_url="http://api.github.com/repos/mysteriumnetwork/node-builds/releases"
else
    releases_url="http://api.github.com/repos/mysteriumnetwork/node/releases"
fi

get_os() {
    local __resultvar=$1
    local result

    result=$(uname | tr '[:upper:]' '[:lower:]')

    eval $__resultvar="'$result'"
}

get_linux_distribution() {
    local __resultvar=$1
    local result

    if [[ -f "/etc/os-release" ]]; then
        local id=$(awk -F= '$1=="ID_LIKE" { print $2 ;}' /etc/os-release)
        if [[ -z "$id" ]]; then
            id=$(awk -F= '$1=="ID" { print $2 ;}' /etc/os-release)
        fi
        result="$id"
    else
        result="unknown"
    fi

    eval $__resultvar="'$result'"
}

get_architecture() {
    local __resultvar=$1
    local result

    if [[ -x "$(command -v dpkg)" ]]; then
        result=$(dpkg --print-architecture)
    else
        result=$(uname -m)
    fi

    if [[ "$result" == "x86_64" ]]; then
        result="amd64"
    fi

    eval $__resultvar="'$result'"
}

download_latest_package() {
    local __resultvar=$1
    local result

    readonly local os=$2
    readonly local arch=$3
    readonly local distro=$4

    local package_file

    if [[ "$distro" == "debian" ]] && [[ "$arch" == "amd64" ]]; then
        package_file="myst_linux_amd64.deb"
    elif [[ "$distro" == "debian" ]] && [[ "$arch" == "armhf" ]]; then
        package_file="myst_linux_armhf.deb"
    fi

    if [[ -z "$package_file" ]]; then
        echo "Error: could not determine package to download, aborting"
        exit 1
    fi

    latest_release_json=$(curl --silent --location "$releases_url/latest")
    latest_tag=$(echo "$latest_release_json" | jq --raw-output '.tag_name')
    echo "Latest release: $latest_tag"

    package_url=$(echo "$latest_release_json" | jq --raw-output --arg package "$package_file" '.assets[] | select(.name==$package) | .browser_download_url')
    echo "Package URL: $package_url"

    curl --write-out '%{http_code}\n' --location "$package_url" --output "$package_file"

    result="$package_file"
    echo $result
    eval $__resultvar="'$result'"
}

install_if_not_exists() {
    dep=$1
    if ! [[ -x "$(command -v $dep)" ]]; then
        echo "$dep not found, installing"
        apt-get -y install "$dep"
    else
        echo "$dep found"
    fi
}

install_script_dependencies() {
    install_if_not_exists curl
    install_if_not_exists libcap2-bin
    install_if_not_exists jq
}

install_dependencies() {
    # Wireguard
    echo "deb http://deb.debian.org/debian/ unstable main" > /etc/apt/sources.list.d/unstable.list
    printf 'Package: *\nPin: release a=unstable\nPin-Priority: 90\n' > /etc/apt/preferences.d/limit-unstable
    apt-get install -y dirmngr
    apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 04EE7237B7D453EC
    apt-key adv --keyserver keyserver.ubuntu.com --recv-keys ECCB6A56B22C536D
    apt-get update
    apt-get install -y wireguard
}

install_myst() {
    readonly local package_file=$1
    apt install -y --allow-downgrades "./$package_file"
    service mysterium-node restart
}

echo "### Installing script dependencies"
install_script_dependencies
echo "### Installing script dependencies - done"

echo "### Detecting platform"
get_os OS
get_architecture ARCH
get_linux_distribution DISTRO
echo "OS: $OS | architecture: $ARCH | distribution: $DISTRO"
echo "### Detecting platform - done"

echo "### Downloading latest package"
download_latest_package PACKAGE_FILE $OS $ARCH $DISTRO
echo "### Downloading latest package - done: $PACKAGE_FILE"

echo "### Installing myst dependencies"
install_dependencies
echo "### Installing myst dependencies - done"

echo "### Installing myst & restarting service"
install_myst $PACKAGE_FILE

echo "### Installation complete!"
