#!/bin/bash -ev

install -m 755 files/myst_linux_armhf.deb	"${ROOTFS_DIR}/tmp/"
install -m 644 files/default-myst-conf "${ROOTFS_DIR}/etc/default/mysterium-node"

on_chroot << EOF
echo "deb http://deb.debian.org/debian/ unstable main" | tee --append /etc/apt/sources.list.d/unstable.list
printf 'Package: *\nPin: release a=unstable\nPin-Priority: 150\n' | tee --append /etc/apt/preferences.d/limit-unstable

apt-get -y install raspberrypi-kernel-headers dirmngr
apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 8B48AD6246925553 7638D0442B90D010 04EE7237B7D453EC

apt-get update
apt-get -y install wireguard openvpn

dpkg -i /tmp/myst_linux_armhf.deb

EOF

rm -f "${ROOTFS_DIR}/tmp/myst_linux_armhf.deb"
