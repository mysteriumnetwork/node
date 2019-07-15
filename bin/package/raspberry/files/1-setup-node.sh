#!/bin/bash -ev

export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
export DEBIAN_FRONTEND="noninteractive"

add_apt_source() {
  local src=$1
  local src_file=$2
  # Check if the source is missing from source file (do not add repeated entries)
  grep -qF "$src" "$src_file" || echo "$src" | tee -a "$src_file"
}

# Enable SSH access
touch /boot/ssh

install --mode=644 default-myst-conf /etc/default/mysterium-node

add_apt_source "deb http://deb.debian.org/debian/ unstable main" "/etc/apt/sources.list.d/unstable.list"
printf 'Package: *\nPin: release a=unstable\nPin-Priority: 150\n' | tee --append /etc/apt/preferences.d/limit-unstable

add_apt_source "deb http://ppa.launchpad.net/mysteriumnetwork/node/ubuntu bionic main" "/etc/apt/sources.list.d/mysterium.list"

apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 8B48AD6246925553 7638D0442B90D010 04EE7237B7D453EC ECCB6A56B22C536D
apt-get update --allow-releaseinfo-change
apt-get -y install \
  wireguard \
  openvpn

apt-get -y install \
  unattended-upgrades

if [[ "${RELEASE_BUILD}" == "true" ]]; then
  echo "Release build, setting up auto-update"
  install --mode=644 unattended-upgrades /etc/apt/apt.conf.d/50unattended-upgrades
fi

chmod 755 myst_linux_armhf.deb
yes | dpkg -i myst_linux_armhf.deb
