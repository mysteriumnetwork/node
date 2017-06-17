#!/bin/bash

function disable_systemd {
    printf  "Disabling systemd script '/lib/systemd/system/mysterium-client.service'..\n" \
        && systemctl disable mysterium-client \
        && rm -f /lib/systemd/system/mysterium-client.service
}

function disable_update_rcd {
    printf  "Disabling initd script '/lib/systemd/system/mysterium-client.service'..\n" \
        && update-rc.d -f mysterium-client remove \
        && rm -f /etc/init.d/mysterium-client
}

function disable_chkconfig {
    printf  "Disabling chkconfig..\n" \
        && chkconfig --del mysterium-client \
        && rm -f /etc/init.d/mysterium-client
}

if [[ -f /etc/redhat-release ]]; then
    # RHEL-variant logic
    if [[ "$1" = "0" ]]; then
	# MysteriumClient is no longer installed, remove from init system
	rm -f /etc/default/mysterium-client
	
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
	rm -f /etc/default/mysterium-client
	
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
	    # MysteriumClient is no longer installed, remove from init system
	    rm -f /etc/default/mysterium-client
	    disable_chkconfig
	fi
    fi
fi