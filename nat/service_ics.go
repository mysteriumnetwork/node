//// +build windows

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

// Enable enables internet connection sharing for the public interface.
func (nat *serviceICS) Enable() error {
	status, err := powerShell("Get-Service RemoteAccess | foreach { $_.StartType }")
	if err != nil {
		return errors.Wrap(err, "failed to get RemoteAccess service startup type")
	}
	nat.remoteAccessStatus = string(status)

	if _, err := powerShell("Set-Service -Name RemoteAccess -StartupType automatic"); err != nil {
		return errors.Wrap(err, "failed to set RemoteAccess service startup type to automatic")
	}
	if _, err := powerShell("Start-Service -Name RemoteAccess"); err != nil {
		return errors.Wrap(err, "failed to start RemoteAccess service")
	}

	ifaceName, err := getPublicInterfaceName()
	if err != nil {
		return errors.Wrap(err, "failed to get public interface name")
	}

	_, err = powerShell(`regsvr32 /s hnetcfg.dll;
		$netShare = New-Object -ComObject HNetCfg.HNetShare;
		$c = $netShare.EnumEveryConnection |? { $netShare.NetConnectionProps.Invoke($_).Name -eq "` + ifaceName + `" };
		$config = $netShare.INetSharingConfigurationForINetConnection.Invoke($c);
		$config.EnableSharing(0)`)

	return errors.Wrap(err, "failed to enable internet connection sharing")
}

// Add enables internet connection sharing for the local interface.
func (nat *serviceICS) Add(rule RuleForwarding) error {
	// TODO firewall rule configuration should be added here for new connections.
	ifaceName, err := getInterfaceBySubnet(rule.SourceAddress)
	if err != nil {
		return errors.Wrap(err, "failed to get public interface name")
	}

	_, err = powerShell(`regsvr32 /s hnetcfg.dll;
		$netShare = New-Object -ComObject HNetCfg.HNetShare;
		$c = $netShare.EnumEveryConnection |? { $netShare.NetConnectionProps.Invoke($_).Name -eq "` + ifaceName + `" };
		$config = $netShare.INetSharingConfigurationForINetConnection.Invoke($c);
		$config.EnableSharing(1)`)

	return errors.Wrap(err, "failed to enable internet connection sharing for internal interface")
}

// Del disables internet connection sharing for the local interface.
func (nat *serviceICS) Del(rule RuleForwarding) error {
	// TODO firewall rule configuration should be added here for cleaning up unused connections.
	ifaceName, err := getInterfaceBySubnet(rule.SourceAddress)
	if err != nil {
		return errors.Wrap(err, "failed to find suitable interface")
	}

	_, err = powerShell(`regsvr32 /s hnetcfg.dll;
		$netShare = New-Object -ComObject HNetCfg.HNetShare;
		$c = $netShare.EnumEveryConnection |? { $netShare.NetConnectionProps.Invoke($_).Name -eq "` + ifaceName + `" };
		$config = $netShare.INetSharingConfigurationForINetConnection.Invoke($c);
		$config.DisableSharing()`)

	return errors.Wrap(err, "failed to disable internet connection sharing for internal interface")
}

// Disable disables internet connection sharing for the public interface.
func (nat *serviceICS) Disable() error {
	// TODO stop internet connection sharing on all enabled interfaces (nat.ifaces).
	// TODO we should cleanup as much as possible, not failing after errors.
	_, err := powerShell("Set-Service -Name RemoteAccess -StartupType " + nat.remoteAccessStatus)
	if err != nil {
		return errors.Wrap(err, "failed to revert RemoteAccess service startup type")
	}

	ifaceName, err := getPublicInterfaceName()
	if err != nil {
		return errors.Wrap(err, "failed to get public interface name")
	}

	_, err = powerShell(`regsvr32 /s hnetcfg.dll;
		$netShare = New-Object -ComObject HNetCfg.HNetShare;
		$c = $netShare.EnumEveryConnection |? { $netShare.NetConnectionProps.Invoke($_).Name -eq "` + ifaceName + `" };
		$config = $netShare.INetSharingConfigurationForINetConnection.Invoke($c);
		$config.DisableSharing()`)

	return errors.Wrap(err, "failed to disable internet connection sharing")
}

func getInterfaceBySubnet(subnet string) (string, error) {
	_, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse subnet from request")
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", errors.Wrap(err, "failed to get a list of network interfaces")
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return "", errors.Wrap(err, "failed to get list of interface addresses")
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ipnet.Contains(ip) {
				return iface.Name, nil
			}
		}
	}
	return "", errors.New("interface not found")
}

func getPublicInterfaceName() (string, error) {
	out, err := powerShell(`Get-WmiObject -Class Win32_IP4RouteTable | where { $_.destination -eq '0.0.0.0' -and $_.mask -eq '0.0.0.0'} | foreach { $_.InterfaceIndex }`)
	if err != nil {
		return "", errors.Wrap(err, "failed to get interface from the default route")
	}

	ifaceID, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return "", errors.Wrap(err, "failed to parse interface ID")
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", errors.Wrap(err, "failed to get a list of network interfaces")
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
