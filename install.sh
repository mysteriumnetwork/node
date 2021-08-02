#!/usr/bin/env bash

shopt -s extglob
set -e
set -o errtrace
set -o errexit
set -o pipefail

export DEBIAN_FRONTEND="noninteractive"

# Install latest release of myst for debian/ubuntu/raspbian
#
# Variables:
# - SNAPSHOT (default: false) - set to "true" to install development snapshot
#

if [[ "$SNAPSHOT" == "true" ]]; then
    PPA="ppa:mysteriumnetwork/node-dev"
    PPA_URL="http://ppa.launchpad.net/mysteriumnetwork/node-dev/ubuntu"
    PPA_FINGER="ECCB6A56B22C536D"
elif [[ "$NETWORK" == "testnet3" ]]; then
    PPA="ppa:mysteriumnetwork/node-testnet3"
    PPA_URL="http://ppa.launchpad.net/mysteriumnetwork/node-testnet3/ubuntu"
    PPA_FINGER="ECCB6A56B22C536D"
else
    PPA="ppa:mysteriumnetwork/node"
    PPA_URL="http://ppa.launchpad.net/mysteriumnetwork/node/ubuntu"
    PPA_FINGER="ECCB6A56B22C536D"
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
        local id=$(awk -F= '$1=="ID" { print $2 ;}' /etc/os-release)
        if [[ -z "$id" ]]; then
            id=$(awk -F= '$1=="ID_LIKE" { print $2 ;}' /etc/os-release)
        fi

        if [[ "$id" == "debian" ]]; then
            if [[ "$(uname -a | grep -c raspberry)" == "1" ]]; then
                id="raspbian"
            fi
        fi

        result="$id"
    else
        result="unknown"
    fi

    eval $__resultvar="'$result'"
}

get_version_codename() {
    local __resultvar=$1
    local result

    if [[ -f "/etc/os-release" ]]; then
        local id=$(awk -F= '$1=="VERSION_CODENAME" { print $2 ;}' /etc/os-release)
        result="$id"
    else
        result="unknown"
    fi

    eval $__resultvar="'$result'"
}



install_ubuntu() {
    apt update

    # add-apt-repository may not be available in Ubuntu server out of the box
    apt install -y software-properties-common

    if [[ "$container" != "docker" ]]; then
        apt install -y "linux-headers-$(uname -r)"
    fi

    # myst
    add-apt-repository -y "$PPA"
    apt update
    apt install -y myst
}


install_debian() {
    # Wireguard
    prepare_sources_list

    if [[ "$container" != "docker" ]]; then
        apt update
        if [[ "$DISTRO" == "raspbian" ]]; then
            apt install -y raspberrypi-kernel-headers
        else
            apt install -y "linux-headers-$(uname -r)"
        fi
    fi

    # myst
    echo "deb $PPA_URL focal main" > /etc/apt/sources.list.d/mysterium.list
    apt-key adv --keyserver keyserver.ubuntu.com --recv-keys "$PPA_FINGER"

    apt update
    apt install -y wireguard myst
}

prepare_sources_list() {
    if [[ "$VERSION_CODENAME" == "buster" ]]; then
      echo "deb http://deb.debian.org/debian ${VERSION_CODENAME}-backports main" > /etc/apt/sources.list.d/backports.list
      apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 04EE7237B7D453EC
      apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 648ACFD622F3D138
    else
      echo "deb http://deb.debian.org/debian/ unstable main" > /etc/apt/sources.list.d/unstable.list
      echo -e "Package: *\nPin: release a=unstable\nPin-Priority: 90" > /etc/apt/preferences.d/limit-unstable
    fi
}

install() {
    case "$DISTRO" in
        ubuntu)
            install_ubuntu
            ;;
        *)
            install_debian
        esac
}


echo "### Detecting platform"
get_os OS
get_linux_distribution DISTRO
get_version_codename VERSION_CODENAME

echo "System info:
OS: $OS
Distribution: $DISTRO
Version codename: $VERSION_CODENAME"
echo "### Detecting platform - done"

echo "### Installing myst & dependencies"
install
echo "### Installing myst & dependencies - done"

echo "### Installation complete!"
