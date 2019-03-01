// +build windows

/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package nat

import (
	"net"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type serviceICS struct {
	ifaces             map[string]struct{} // list in internal interfaces with enabled internet connection sharing
	remoteAccessStatus string
}

func (nat *serviceICS) Enable() error {
	status, err := powerShell("Get-Service RemoteAccess | foreach { $_.StartType }")
	if err != nil {
		return err
	}
	nat.remoteAccessStatus = string(status)

	if _, err := powerShell("Set-Service -Name RemoteAccess -StartupType automatic"); err != nil {
		return err
	}
	if _, err := powerShell("Start-Service -Name RemoteAccess"); err != nil {
		return err
	}

	ifaceName, err := getPublicInterfaceName()
	if err != nil {
		return err
	}

	// Enable internet connection sharing for the public interface.
	_, err = powerShell(`regsvr32 /s hnetcfg.dll;
		$netShare = New-Object -ComObject HNetCfg.HNetShare;
		$c = $netShare.EnumEveryConnection |? { $netShare.NetConnectionProps.Invoke($_).Name -eq "` + ifaceName + `" };
		$config = $netShare.INetSharingConfigurationForINetConnection.Invoke($c);
		$config.EnableSharing(0)`)

	return err
}

func (nat *serviceICS) Add(rule RuleForwarding) error {
	// Enable internet connection sharing for the local interface.
	// TODO detect interface name from the rule.
	// TODO firewall rule configuration should be added here for new connections.
	ifaceName := "myst1"

	_, err := powerShell(`regsvr32 /s hnetcfg.dll;
	$netShare = New-Object -ComObject HNetCfg.HNetShare;
	$c = $netShare.EnumEveryConnection |? { $netShare.NetConnectionProps.Invoke($_).Name -eq "` + ifaceName + `" };
	$config = $netShare.INetSharingConfigurationForINetConnection.Invoke($c);
	$config.EnableSharing(1)`)

	return err
}

func (nat *serviceICS) Del(rule RuleForwarding) error {
	// Disable internet connection sharing for the local interface.
	// TODO detect interace name from the rule.
	// TODO firewall rule configuration should be added here for cleaning up unused connections.
	ifaceName := "myst1"

	_, err := powerShell(`regsvr32 /s hnetcfg.dll;
		$netShare = New-Object -ComObject HNetCfg.HNetShare;
		$c = $netShare.EnumEveryConnection |? { $netShare.NetConnectionProps.Invoke($_).Name -eq "` + ifaceName + `" };
		$config = $netShare.INetSharingConfigurationForINetConnection.Invoke($c);
		$config.DisableSharing()`)

	return err
}

func (nat *serviceICS) Disable() error {
	// TODO stop internet connection sharing on all enabled interfaces (nat.ifaces).
	// TODO we should cleanup as much as possible, not failing after errors.
	_, err := powerShell("Set-Service -Name RemoteAccess -StartupType " + nat.remoteAccessStatus)
	if err != nil {
		return err
	}

	ifaceName, err := getPublicInterfaceName()
	if err != nil {
		return err
	}

	// Disable internet connection sharing for the public interface.
	_, err = powerShell(`regsvr32 /s hnetcfg.dll;
		$netShare = New-Object -ComObject HNetCfg.HNetShare;
		$c = $netShare.EnumEveryConnection |? { $netShare.NetConnectionProps.Invoke($_).Name -eq "` + ifaceName + `" };
		$config = $netShare.INetSharingConfigurationForINetConnection.Invoke($c);
		$config.DisableSharing()`)

	return err
}

func getPublicInterfaceName() (string, error) {
	out, err := powerShell(`Get-WmiObject -Class Win32_IP4RouteTable | where { $_.destination -eq '0.0.0.0' -and $_.mask -eq '0.0.0.0'} | foreach { $_.InterfaceIndex }`)
	if err != nil {
		return "", err
	}

	ifaceID, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return "", err

	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		if iface.Index == ifaceID {
			return iface.Name, nil
		}
	}

	return "", errors.New("interface not found")
}

func powerShell(cmd string) ([]byte, error) {
	out, err := exec.Command("powershell", "-Command", cmd).CombinedOutput()
	return out, errors.Wrap(err, string(out))
}
