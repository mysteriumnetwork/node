#!/bin/bash

function disable_systemd {
    printf  "Disabling systemd script '/lib/systemd/system/mysterium-node.service'..\n"
    systemctl disable mysterium-node
    rm -f /lib/systemd/system/mysterium-node.service
}

function disable_update_rcd {
    printf  "Disabling initd script '/lib/systemd/system/mysterium-node.service'..\n"
    update-rc.d -f mysterium-node remove
    rm -f /etc/init.d/mysterium-node
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
