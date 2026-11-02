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
    PPA_URL="https://ppa.mysterium.network/mysteriumnetwork/node-dev/ubuntu"
elif [[ "$NETWORK" == "testnet3" ]]; then
    PPA="ppa:mysteriumnetwork/node-testnet3"
    PPA_URL="https://ppa.mysterium.network/mysteriumnetwork/node-testnet3/ubuntu"
else
    PPA="ppa:mysteriumnetwork/node"
    PPA_URL="https://ppa.mysterium.network/mysteriumnetwork/node/ubuntu"
fi

# The official installer uses this single signing-key endpoint for stable,
# snapshot and testnet3 channels. The fingerprint is verified before use.
PPA_KEY_URL="https://ppa.mysterium.network/public-key"
PPA_KEY_FINGERPRINT="BC7F93152681BDE3356F4080A197201433627AB4"
APT_KEYRING_DIR="${APT_KEYRING_DIR:-/etc/apt/keyrings}"
APT_SOURCE_DIR="${APT_SOURCE_DIR:-/etc/apt/sources.list.d}"

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
    # Configure the authenticated Mysterium source before the first refresh.
    configure_mysterium_repository

    apt update

    if [[ "$container" != "docker" ]]; then
        apt install -y "linux-headers-$(uname -r)"
    fi

    apt install -y myst
}


install_debian() {
    # Wireguard
    prepare_sources_list

    # myst - set up key and repo before any apt update to avoid stale list issues
    configure_mysterium_repository

    if [[ "$container" != "docker" ]]; then
        apt update
        if [[ "$DISTRO" == "raspbian" ]]; then
            apt install -y raspberrypi-kernel-headers
        else
            apt install -y "linux-headers-$(uname -r)"
        fi
    fi

    apt update
    apt install -y wireguard myst
}

configure_mysterium_repository() {
    local key_temp keyring_temp key_stage source_stage fingerprint
    key_temp=$(mktemp) || return 1
    keyring_temp=$(mktemp) || { rm -f "$key_temp"; return 1; }
    trap 'rm -f "${key_temp:-}" "${keyring_temp:-}" "${key_stage:-}" "${source_stage:-}"' RETURN

    if ! curl -fsSL --proto '=https' --tlsv1.2 \
        "$PPA_KEY_URL" -o "$key_temp"; then
        echo "Failed to download the official Mysterium signing key" >&2
        return 1
    fi
    fingerprint=$(gpg --batch --with-colons --import-options show-only \
        --import "$key_temp" | awk -F: -v expected="$PPA_KEY_FINGERPRINT" '
			$1 == "pub" { primary++; in_primary=1; next }
			$1 == "sub" { in_primary=0; next }
            $1 == "sec" || $1 == "ssb" { secret++ }
			$1 == "fpr" && in_primary && $10 == expected { matched++; in_primary=0 }
            END { if (primary != 1 || secret != 0 || matched != 1) exit 1; print expected }
        ') || {
        echo "Failed to inspect the Mysterium signing key" >&2
        return 1
    }
    if [[ "$fingerprint" != "$PPA_KEY_FINGERPRINT" ]]; then
        echo "Mysterium signing key fingerprint mismatch" >&2
        return 1
    fi
    if ! gpg --batch --yes --dearmor -o "$keyring_temp" "$key_temp"; then
        echo "Failed to convert the Mysterium signing key" >&2
        return 1
    fi
    mkdir -p "$APT_KEYRING_DIR" || return 1
    mkdir -p "$APT_SOURCE_DIR" || return 1
    key_stage=$(mktemp "$APT_KEYRING_DIR/.mysterium.gpg.XXXXXX") || return 1
    source_stage=$(mktemp "$APT_SOURCE_DIR/.mysterium.list.XXXXXX") || return 1
    chmod 0644 "$key_stage" "$source_stage"
    if ! install -o root -g root -m 0644 "$keyring_temp" "$key_stage"; then
        echo "Failed to stage the Mysterium keyring" >&2
        return 1
    fi
	if [[ -e "$APT_SOURCE_DIR/mysterium.list" ]]; then
		if [[ ! -f "$APT_SOURCE_DIR/mysterium.list" || ! -r "$APT_SOURCE_DIR/mysterium.list" ]]; then
			echo "Cannot safely inspect the existing Mysterium repository definition" >&2
			return 1
		fi
		# Remove only the three official Mysterium channels, including legacy
		# Launchpad URLs. Other Launchpad repositories remain untouched.
		awk '!/https?:\/\/(ppa\.mysterium\.network|ppa\.launchpad\.net)\/mysteriumnetwork\/node(-dev|-testnet3)?\/ubuntu/ { print }' \
			"$APT_SOURCE_DIR/mysterium.list" > "$source_stage" || return 1
	fi
	printf 'deb [signed-by=%s/mysterium.gpg] %s focal main\n' "$APT_KEYRING_DIR" "$PPA_URL" >> "$source_stage" || return 1
    mv -f "$key_stage" "$APT_KEYRING_DIR/mysterium.gpg" || {
        echo "Failed to activate the Mysterium keyring; repository was not changed" >&2
        return 1
    }
    mv -f "$source_stage" "$APT_SOURCE_DIR/mysterium.list" || {
        echo "Failed to activate the Mysterium repository definition; keyring was updated, retry is required" >&2
        return 1
    }
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

install_node() {
    case "$DISTRO" in
        ubuntu)
            install_ubuntu
            ;;
        *)
            install_debian
        esac
}

if [[ "${MYSTERIUM_INSTALLER_LIBRARY:-0}" != "1" ]]; then
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
install_node
echo "### Installing myst & dependencies - done"

echo "### Installation complete!"
fi
