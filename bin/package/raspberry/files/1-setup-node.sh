#!/bin/bash -v

export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
export DEBIAN_FRONTEND="noninteractive"

# Adds APT source once (no repeated entries)
add_apt_source() {
  local src=$1
  local src_file=$2
  grep -qF "$src" "$src_file" || echo "$src" | tee -a "$src_file"
}

# Enable SSH access
touch /boot/ssh

# Add APT sources
add_apt_source "deb http://ppa.launchpad.net/mysteriumnetwork/node/ubuntu focal main" "/etc/apt/sources.list.d/mysterium.list"
apt-key adv --keyserver keyserver.ubuntu.com --recv-keys ECCB6A56B22C536D

add_apt_source "deb http://deb.debian.org/debian/ unstable main" "/etc/apt/sources.list.d/unstable.list"
wget -O - https://ftp-master.debian.org/keys/archive-key-$(lsb_release -sr).asc | sudo apt-key add -
printf 'Package: *\nPin: release a=unstable\nPin-Priority: 150\n' | sudo tee --append /etc/apt/preferences.d/limit-unstable

# Import missing keys
apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 0E98404D386FA1D9
apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 6ED0E7B82643E131

apt-get update --allow-releaseinfo-change

# Install myst dependencies
apt-get -y install \
  wireguard \
  openvpn

# Setup unattended upgrades
apt-get -y install \
  unattended-upgrades
if [[ "${RELEASE_BUILD}" == "true" ]]; then
  echo "Release build, setting up auto-update"
  install --mode=644 unattended-upgrades /etc/apt/apt.conf.d/50unattended-upgrades
  install --mode=644 auto-upgrades /etc/apt/apt.conf.d/20auto-upgrades
fi

# Install myst
install --mode=644 default-myst-conf /etc/default/mysterium-node
mkdir -p /etc/mysterium-node
install --mode=644 config.toml /etc/mysterium-node/config.toml
chmod 755 myst_linux_armhf.deb
yes | dpkg --install --force-depends myst_linux_armhf.deb
chown -R mysterium-node:mysterium-node /etc/mysterium-node
