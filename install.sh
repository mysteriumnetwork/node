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

get_matching_ubuntu_codename() {
    local __resultvar=$1
    local result
    readonly local mdistro=$2
    readonly local mvcodename=$3

    if [[ "$mdistro" == "debian" || "$mdistro" == "raspbian" ]]; then
        case "$mvcodename" in
        eoan)
            result="eoan"
            ;;
        buster)
            result="bionic"
            ;;
        stretch)
            result="xenial"
            ;;
        *)
            result="bionic"
        esac
    else
        result="bionic"
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

install_if_not_exists() {
    dep=$1
    if ! [[ -x "$(command -v $dep)" ]]; then
        echo "$dep not found, installing"
        apt -y install "$dep"
    else
        echo "$dep found"
    fi
}

install_script_dependencies() {
    apt update
    install_if_not_exists curl
    install_if_not_exists libcap2-bin
    install_if_not_exists jq
    apt install -y dirmngr
}

install_ubuntu() {
    # openvpn, etc.
    if [[ "$VERSION_CODENAME" == "xenial" ]]; then
        curl -s https://swupdate.openvpn.net/repos/repo-public.gpg | apt-key add -
        echo "deb http://build.openvpn.net/debian/openvpn/stable xenial main" > /etc/apt/sources.list.d/openvpn-aptrepo.list
    fi
    apt update
    apt install -y ipset resolvconf openvpn

    # add-apt-repository may not be available in Ubuntu server out of the box
    apt install -y software-properties-common

    # Wireguard
    # If kernel module installs successfully, the following commands should give no errors:
    # ip link add dev wg0 type wireguard
    # ip link delete wg0
    if [[ "$container" != "docker" ]]; then
        apt install -y "linux-headers-$(uname -r)"
    fi
    add-apt-repository -y ppa:wireguard/wireguard
    apt update
    apt install -y wireguard
    dpkg-reconfigure wireguard-dkms

    # myst
    add-apt-repository -y "$PPA"
    apt update
    apt-get --only-upgrade install -y myst
    apt install -y myst
    service mysterium-node restart
}

install_raspbian() {
    # openvpn, etc.
    apt update
    apt install -y ipset resolvconf openvpn

    # Wireguard
    apt install -y git bc bison flex libssl-dev libncurses5-dev # For rpi-source
    wget https://raw.githubusercontent.com/notro/rpi-source/master/rpi-source -O /usr/local/bin/rpi-source \
        && chmod +x /usr/local/bin/rpi-source \
        && rpi-source -q --tag-update
    rpi-source --default-config || true

    echo "deb http://ppa.launchpad.net/wireguard/wireguard/ubuntu $UBUNTU_VERSION_CODENAME main" > /etc/apt/sources.list.d/wireguard.list
    apt-key adv --keyserver keyserver.ubuntu.com --recv-keys AE33835F504A1A25
    apt update
    apt install -y wireguard
    dpkg-reconfigure wireguard-dkms

    # myst
    echo "deb $PPA_URL $UBUNTU_VERSION_CODENAME main" > /etc/apt/sources.list.d/mysterium.list
    apt-key adv --keyserver keyserver.ubuntu.com --recv-keys "$PPA_FINGER"
    apt update
    apt-get --only-upgrade install -y myst
    apt install -y myst
    service mysterium-node restart
}

install_debian() {
    # openvpn, etc.
    apt update
    apt install -y ipset resolvconf openvpn

    # Wireguard
    echo "deb http://deb.debian.org/debian/ unstable main" > /etc/apt/sources.list.d/unstable.list
    printf 'Package: *\nPin: release a=unstable\nPin-Priority: 90\n' > /etc/apt/preferences.d/limit-unstable
    apt update
    if [[ "$container" != "docker" ]]; then
        apt install -y "linux-headers-$(uname -r)"
    fi
    apt install -y wireguard
    dpkg-reconfigure wireguard-dkms

    # myst
    echo "deb $PPA_URL $UBUNTU_VERSION_CODENAME main" > /etc/apt/sources.list.d/mysterium.list
    apt-key adv --keyserver keyserver.ubuntu.com --recv-keys "$PPA_FINGER"
    apt update
    apt-get --only-upgrade install -y myst
    apt install -y myst
    service mysterium-node restart
}

install() {
    case "$DISTRO" in
        ubuntu)
            install_ubuntu
            ;;
        raspbian)
            install_raspbian
            ;;
        *)
            install_debian
        esac
}

echo "### Installing script dependencies"
install_script_dependencies
echo "### Installing script dependencies - done"

echo "### Detecting platform"
get_os OS
get_architecture ARCH
get_linux_distribution DISTRO
get_version_codename VERSION_CODENAME
get_matching_ubuntu_codename UBUNTU_VERSION_CODENAME "$DISTRO" "$VERSION_CODENAME"

echo "System info:
OS: $OS
Architecture: $ARCH
Distribution: $DISTRO
Version codename: $VERSION_CODENAME
Matching ubuntu version codename: $UBUNTU_VERSION_CODENAME"
echo "### Detecting platform - done"

echo "### Installing myst & dependencies"
install
echo "### Installing myst & dependencies - done"

echo "### Installation complete!"
