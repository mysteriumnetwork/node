#!/bin/bash

. /usr/share/debconf/confmodule

OS_DIR_BIN="/usr/bin"
OS_DIR_CONFIG="/etc/mysterium-node"
OS_DIR_LOG="/var/log/mysterium-node"
OS_DIR_RUN="/var/run/mysterium-node"
OS_DIR_DATA="/var/lib/mysterium-node"
OS_DIR_INSTALLATION="/usr/lib/mysterium-node/installation"
OS_DIR_INITD="/etc/init.d/"
OS_DIR_SYSTEMD="/lib/systemd/system"

DAEMON_USER=mysterium-node
DAEMON_GROUP=mysterium-node
DAEMON_DEFAULT=/etc/default/mysterium-node

function install_initd {
    printf "Installing initd script '$OS_DIR_INITD/mysterium-node'..\n" \
        && cp -f $OS_DIR_INSTALLATION/initd.sh $OS_DIR_INITD/mysterium-node \
        && chmod +x $OS_DIR_INITD/mysterium-node
}

function install_systemd {
    printf "Installing systemd script '$OS_DIR_SYSTEMD/mysterium-node.service'..\n" \
        && cp -f $OS_DIR_INSTALLATION/mysterium-node.service $OS_DIR_SYSTEMD/mysterium-node.service \
        && cp -f $OS_DIR_INSTALLATION/mysterium-consumer.service $OS_DIR_SYSTEMD/mysterium-consumer.service \
        && systemctl enable systemd-networkd.service \
        && systemctl enable mysterium-node \
        && systemctl restart mysterium-node
}

function install_update_rcd {
    printf "Installing rc.d config..\n" \
        && update-rc.d mysterium-node defaults
}

function install_chkconfig {
    printf "Installing chkconfig..\n" \
        && chkconfig --add mysterium-node
}

function ensure_paths {
    iptables_path=`which iptables`

    # validate utility against valid system paths
    basepath=${iptables_path%/*}
    echo "iptables basepath detected: ${basepath}"
    if ! [[ ${basepath} =~ (^/usr/sbin|^/sbin|^/bin|^/usr/bin) ]]; then
      echo "invalid basepath for dependency - check if system PATH has not been altered"
      exit 1
    fi

    iptables_required_path="/usr/sbin/iptables"

    if ! [[ -x ${iptables_required_path} ]]; then
        ln -s ${iptables_path} ${iptables_required_path}
    fi
}

printf "Creating user '$DAEMON_USER:$DAEMON_GROUP'...\n" \
    && useradd --system -U $DAEMON_USER -G root -s /bin/false -m -d $OS_DIR_DATA \
    && usermod -a -G root $DAEMON_USER \
    && chown -R -L $DAEMON_USER:$DAEMON_GROUP $OS_DIR_DATA

printf "Creating directories...\n" \
    && mkdir -p $OS_DIR_LOG $OS_DIR_CONFIG $OS_DIR_RUN $OS_DIR_DATA \
    && chown $DAEMON_USER:$DAEMON_GROUP $OS_DIR_LOG $OS_DIR_CONFIG $OS_DIR_RUN $OS_DIR_DATA

printf "Setting required capabilities...\n" \
    && setcap cap_net_admin+ep /usr/bin/myst

ensure_paths

# Add defaults file, if it doesn't exist
if [[ ! -f $DAEMON_DEFAULT ]]; then
    cp $OS_DIR_INSTALLATION/default $DAEMON_DEFAULT
fi

# TODO remove temporary fix for starting all services instead of wireguard.
sed -i 's/^SERVICE_OPTS="wireguard"$/SERVICE_OPTS=""/g' $DAEMON_DEFAULT

printf "\nInstallation successfully finished.\n" \
    && printf "Usage: service mysterium-node restart\n"

# Distribution-specific logic
if [[ -f /etc/redhat-release ]]; then
    # RHEL-variant logic
    which systemctl &>/dev/null
    if [[ $? -eq 0 ]]; then
    	install_systemd || echo "got an error, ignoring - probably systemd-spawn issue"
    else
	    # Assuming sysv
	    install_initd
	    install_chkconfig
    fi
elif [[ -f /etc/debian_version ]]; then
    # Debian/Ubuntu logic
    which systemctl &>/dev/null
    if [[ $? -eq 0 ]]; then
    	install_systemd || echo "got an error, ignoring - probably systemd-spawn issue"
    else
	    # Assuming sysv
    	install_initd
    	install_update_rcd
    fi
elif [[ -f /etc/os-release ]]; then
    source /etc/os-release
    if [[ $ID = "amzn" ]]; then
    	# Amazon Linux logic
    	install_initd
    	install_chkconfig
    fi
fi
