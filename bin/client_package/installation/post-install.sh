#!/bin/bash

OS_DIR_BIN="/usr/bin"
OS_DIR_LOG="/var/log/mysterium-client"
OS_DIR_RUN="/var/run/mysterium-client"
OS_DIR_DATA="/var/lib/mysterium-client"
OS_DIR_INSTALLATION="/usr/lib/mysterium-client/installation"
OS_DIR_INITD="/etc/init.d/"
OS_DIR_SYSTEMD="/lib/systemd/system"

DAEMON_USER=mysterium-client
DAEMON_GROUP=mysterium-client
DAEMON_DEFAULT=/etc/default/mysterium-client

function install_initd {
    printf "Installing initd script '$OS_DIR_INITD/mysterium-client'..\n" \
        && cp -f $OS_DIR_INSTALLATION/initd.sh $OS_DIR_INITD/mysterium-client \
        && chmod +x $OS_DIR_INITD/mysterium-client
}

function install_systemd {
    printf "Installing systemd script '$OS_DIR_SYSTEMD/mysterium-client.service'..\n" \
        && cp -f $OS_DIR_INSTALLATION/systemd.service $OS_DIR_SYSTEMD/mysterium-client.service \
        && systemctl enable mysterium-client
}

function install_update_rcd {
    printf "Installing rc.d config..\n" \
        && update-rc.d mysterium-client defaults
}

function install_chkconfig {
    printf "Installing chkconfig..\n" \
        && chkconfig --add mysterium-client
}

printf "Creating user '$DAEMON_USER:$DAEMON_GROUP'...\n" \
    && useradd --system -U $DAEMON_USER -G root -s /bin/false -m -d $OS_DIR_DATA \
    && usermod -a -G root $DAEMON_USER \
    && chown -R -L $DAEMON_USER:$DAEMON_GROUP $OS_DIR_DATA

# Remove legacy symlink, if it exists
if [[ -L $OS_DIR_INITD/mysterium-client ]]; then
    rm -f $OS_DIR_INITD/mysterium-client
fi

# Distribution-specific logic
if [[ -f /etc/redhat-release ]]; then
    # RHEL-variant logic
    which systemctl &>/dev/null
    if [[ $? -eq 0 ]]; then
	install_systemd
    else
	# Assuming sysv
	install_initd
	install_chkconfig
    fi
elif [[ -f /etc/debian_version ]]; then
    # Debian/Ubuntu logic
    which systemctl &>/dev/null
    if [[ $? -eq 0 ]]; then
	install_systemd
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

# Add defaults file, if it doesn't exist
if [[ ! -f $DAEMON_DEFAULT ]]; then
    cp $OS_DIR_INSTALLATION/default $DAEMON_DEFAULT
fi

printf "\nInstallation successfully finished.\n" \
    && printf "Usage: service mysterium-client restart\n"