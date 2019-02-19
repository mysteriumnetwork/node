#!/bin/bash

if [ "$1" = "purge" ]; then
    if [[ -e /usr/share/debconf/confmodule ]]; then
        # Source debconf library.
        . /usr/share/debconf/confmodule
        # Remove my changes to the db.
        db_purge
    else
        printf  "confmodule is missing, debconf db data was not purged..\n"
    fi
fi

function disable_systemd {
    system_service=/lib/systemd/system/mysterium-node.service
    if [ ! -e $system_service ]; then
        return
    fi
    printf  "Disabling systemd script '$system_service'..\n"
    systemctl disable mysterium-node
    rm -f $system_service
}

function disable_update_rcd {
    initd=/etc/init.d/mysterium-node
    printf  "Disabling initd script '$initd'..\n"
    update-rc.d -f mysterium-node remove
    rm -f $initd
}

function disable_chkconfig {
    printf  "Disabling chkconfig..\n"
    chkconfig --del mysterium-node
    rm -f /etc/init.d/mysterium-node
}

if [[ -f /etc/redhat-release ]]; then
    # RHEL-variant logic
    if [[ "$1" = "0" ]]; then
	# MysteriumNode is no longer installed, remove from init system
	rm -f /etc/default/mysterium-node
	
	which systemctl &>/dev/null
	if [[ $? -eq 0 ]]; then
	    disable_systemd
	else
	    # Assuming sysv
	    disable_chkconfig
	fi
    fi
elif [[ -f /etc/lsb-release ]]; then
    # Debian/Ubuntu logic
    if [[ "$1" != "upgrade" ]]; then
	# Remove/purge
	rm -f /etc/default/mysterium-node
	
	which systemctl &>/dev/null
	if [[ $? -eq 0 ]]; then
	    disable_systemd
	else
	    # Assuming sysv
	    disable_update_rcd
	fi
    fi
elif [[ -f /etc/os-release ]]; then
    source /etc/os-release
    if [[ $ID = "amzn" ]]; then
	# Amazon Linux logic
	if [[ "$1" = "0" ]]; then
	    # MysteriumNode is no longer installed, remove from init system
	    rm -f /etc/default/mysterium-node
	    disable_chkconfig
	fi
    fi
fi
