#!/bin/bash -ev

export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
export DEBIAN_FRONTEND="noninteractive"

systemctl enable ssh
sed -i -e 's/#PasswordAuthentication.*/PasswordAuthentication yes/g' /etc/ssh/sshd_config

install -m 644 default-myst-conf /etc/default/mysterium-node

if [[ "${RELEASE_BUILD}" == "true" ]]; then
  echo "Release build, setting up auto-update"
  install -m 644 unattended-upgrades /etc/apt/apt.conf.d/51unattended-upgrades-myst
fi

echo "deb http://deb.debian.org/debian/ unstable main" | tee --append /etc/apt/sources.list.d/unstable.list
echo "deb http://ppa.launchpad.net/mysteriumnetwork/node/ubuntu bionic main " | tee --append /etc/apt/sources.list.d/mysterium.list
printf 'Package: *\nPin: release a=unstable\nPin-Priority: 150\n' | tee --append /etc/apt/preferences.d/limit-unstable

apt-get -y install raspberrypi-kernel-headers dirmngr
apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 8B48AD6246925553 7638D0442B90D010 04EE7237B7D453EC ECCB6A56B22C536D
apt-get update
apt-get -y install wireguard openvpn

chmod 755 myst_linux_armhf.deb
yes | dpkg -i myst_linux_armhf.deb
