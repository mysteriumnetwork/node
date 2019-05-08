#!/bin/bash -e

install -m 755 files/myst_linux_armhf.deb	"${ROOTFS_DIR}/tmp/"

on_chroot << EOF
echo "deb http://deb.debian.org/debian/ unstable main" | tee --append /etc/apt/sources.list.d/unstable.list
printf 'Package: *\nPin: release a=unstable\nPin-Priority: 150\n' | tee --append /etc/apt/preferences.d/limit-unstable

apt-get -y install raspberrypi-kernel-headers dirmngr
apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 8B48AD6246925553 7638D0442B90D010 04EE7237B7D453EC

apt-get update
apt-get -y install wireguard openvpn

cat <<'EOF' > /etc/default/mysterium-node
# Define additional args for `myst` service (see `myst --help` for full list)
CONF_DIR="--config-dir=/etc/mysterium-node"
RUN_DIR="--runtime-dir=/var/run/mysterium-node"
DATA_DIR="--data-dir=/var/lib/mysterium-node"
DAEMON_OPTS="--tequilapi.address=0.0.0.0"
SERVICE_OPTS="openvpn"
EOF
dpkg -i /tmp/myst_linux_armhf.deb

EOF

rm -f "${ROOTFS_DIR}/tmp/myst_linux_armhf.deb"
