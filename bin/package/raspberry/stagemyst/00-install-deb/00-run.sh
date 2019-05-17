#!/bin/bash -ev

install -m 755 files/myst_linux_armhf.deb	"${ROOTFS_DIR}/tmp/"
install -m 644 files/default-myst-conf "${ROOTFS_DIR}/etc/default/mysterium-node"

if [ "${BUILD_VERSION}" != "${BUILD_DEV_RELEASE}" ]; then
  install -m 644 files/unattended-upgrades "${ROOTFS_DIR}/etc/apt/apt.conf.d/51unattended-upgrades-myst"
fi

on_chroot << EOF
echo "deb http://deb.debian.org/debian/ unstable main" | tee --append /etc/apt/sources.list.d/unstable.list
echo "deb http://ppa.launchpad.net/mysteriumnetwork/node/ubuntu bionic main " | tee --append /etc/apt/sources.list.d/mysterium.list
printf 'Package: *\nPin: release a=unstable\nPin-Priority: 150\n' | tee --append /etc/apt/preferences.d/limit-unstable

apt-get -y install raspberrypi-kernel-headers dirmngr
apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 8B48AD6246925553 7638D0442B90D010 04EE7237B7D453EC ECCB6A56B22C536D

apt-get update
apt-get -y install wireguard openvpn

dpkg -i /tmp/myst_linux_armhf.deb

EOF

rm -f "${ROOTFS_DIR}/tmp/myst_linux_armhf.deb"
