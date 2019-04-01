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
	"strconv"
	"strings"
	"sync"

	log "github.com/cihub/seelog"
	"github.com/pkg/errors"
)

const (
	enablePublicSharing  = "$config.EnableSharing(0)"
	enablePrivateSharing = "$config.EnableSharing(1)"
	disableSharing       = "$config.DisableSharing()"
)

type serviceICS struct {
	mu                 sync.Mutex
	ifaces             map[string]RuleForwarding // list in internal interfaces with enabled internet connection sharing
	remoteAccessStatus string
	powerShell         func(cmd string) ([]byte, error)
	setICSAddresses    func(config map[string]string) (map[string]string, error)
	oldICSConfig       map[string]string
}

// Enable enables internet connection sharing for the public interface.
func (ics *serviceICS) Enable() error {
	if err := ics.enableRemoteAccessService(); err != nil {
		return errors.Wrap(err, "failed to Enable RemoteAccess service")
	}

	ifaceName, err := ics.getPublicInterfaceName()
	if err != nil {
		return errors.Wrap(err, "failed to get public interface name")
	}

	err = ics.applySharingConfig(enablePublicSharing, ifaceName)
	return errors.Wrap(err, "failed to enable internet connection sharing")
}

func (ics *serviceICS) enableRemoteAccessService() error {
	status, err := ics.powerShell("Get-Service RemoteAccess | foreach { $_.StartType }")
	if err != nil {
		return errors.Wrap(err, "failed to get RemoteAccess service startup type")
	}
	ics.remoteAccessStatus = string(status)

	if _, err := ics.powerShell("Set-Service -Name RemoteAccess -StartupType automatic"); err != nil {
		return errors.Wrap(err, "failed to set RemoteAccess service startup type to automatic")
	}

	_, err = ics.powerShell("Start-Service -Name RemoteAccess")
	return errors.Wrap(err, "failed to start RemoteAccess service")
}

// Add enables internet connection sharing for the local interface.
func (ics *serviceICS) Add(rule RuleForwarding) error {
	// TODO firewall rule configuration should be added here for new connections.
	_, ipnet, err := net.ParseCIDR(rule.SourceAddress)
	if err != nil {
		log.Warnf("%s Failed to parse IP-address: %s", natLogPrefix, rule.SourceAddress)
	}

	ip := incrementIP(ipnet.IP)
	ics.oldICSConfig, err = ics.setICSAddresses(map[string]string{
		"ScopeAddress":          ip.String(),
		"ScopeAddressBackup":    ip.String(),
		"StandaloneDhcpAddress": ip.String()})
	if err != nil {
		return errors.Wrap(err, "failed to set ICS IP-address range")
	}

	ifaceName, err := ics.getInternalInterfaceName()
	if err != nil {
		return errors.Wrap(err, "failed to find suitable interface")
	}

	err = ics.applySharingConfig(enablePrivateSharing, ifaceName)
	if err != nil {
		return errors.Wrap(err, "failed to enable internet connection sharing for internal interface")
	}

	ics.mu.Lock()
	defer ics.mu.Unlock()
	ics.ifaces[ifaceName] = rule

	return nil
}

// Del disables internet connection sharing for the local interface.
func (ics *serviceICS) Del(rule RuleForwarding) error {
	// TODO firewall rule configuration should be added here for cleaning up unused connections.
	ifaceName, err := ics.getInternalInterfaceName()
	if err != nil {
		return errors.Wrap(err, "failed to find suitable interface")
	}

	err = ics.applySharingConfig(disableSharing, ifaceName)
	if err != nil {
		return errors.Wrap(err, "failed to disable internet connection sharing for internal interface")
	}

	ics.mu.Lock()
	defer ics.mu.Unlock()
	delete(ics.ifaces, ifaceName)

	return nil
}

// Disable disables internet connection sharing for the public interface.
func (ics *serviceICS) Disable() (resErr error) {
	if _, err := ics.setICSAddresses(ics.oldICSConfig); err != nil {
		return errors.Wrap(err, "failed to revert ICS IP-address range")
	}
	for iface, rule := range ics.ifaces {
		if err := ics.Del(rule); err != nil {
			log.Errorf("%s Failed to cleanup internet connection sharing for '%s' interface: %v", natLogPrefix, iface, err)
			if resErr == nil {
				resErr = err
			}
		}
	}

	_, err := ics.powerShell("Set-Service -Name RemoteAccess -StartupType " + ics.remoteAccessStatus)
	if err != nil {
		err = errors.Wrap(err, "failed to revert RemoteAccess service startup type")
		log.Errorf("%s %v", natLogPrefix, err)
		if resErr == nil {
			resErr = err
		}
	}

	ifaceName, err := ics.getPublicInterfaceName()
	if err != nil {
		err = errors.Wrap(err, "failed to get public interface name")
		log.Errorf("%s %v", natLogPrefix, err)
		if resErr == nil {
			resErr = err
		}
	}

	err = ics.applySharingConfig(disableSharing, ifaceName)
	if err != nil {
		err = errors.Wrap(err, "failed to disable internet connection sharing")
		log.Errorf("%s %v", natLogPrefix, err)
		if resErr == nil {
			resErr = err
		}
	}

	return resErr
}

func (ics *serviceICS) getPublicInterfaceName() (string, error) {
	out, err := ics.powerShell(`Get-WmiObject -Class Win32_IP4RouteTable | where { $_.destination -eq '0.0.0.0' -and $_.mask -eq '0.0.0.0'} | foreach { $_.InterfaceIndex }`)
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

func (ics *serviceICS) applySharingConfig(action, ifaceName string) error {
	if len(action) == 0 {
		return errors.New("empty action provided")
	}
	if len(ifaceName) == 0 {
		return errors.New("empty interface name provided")
	}

	_, err := ics.powerShell(`regsvr32 /s hnetcfg.dll;
		$netShare = New-Object -ComObject HNetCfg.HNetShare;
		$c = $netShare.EnumEveryConnection |? { $netShare.NetConnectionProps.Invoke($_).Name -eq "` + ifaceName + `" };
		$config = $netShare.INetSharingConfigurationForINetConnection.Invoke($c);` + action)
	return err
}

func (ics *serviceICS) getInternalInterfaceName() (string, error) {
	out, err := ics.powerShell(`Get-WmiObject Win32_NetworkAdapter | Where-Object {$_.ServiceName -eq "tap0901"} | foreach { $_.NetConnectionID }`)
	if err != nil {
		return "", errors.Wrap(err, "failed to detect internal interface name")
	}

	ifaceName := strings.TrimSpace(string(out))
	if len(ifaceName) == 0 {
		return "", errors.New("interface not found")
	}

	return ifaceName, nil
}

func incrementIP(ip net.IP) net.IP {
	dup := make(net.IP, len(ip))
	copy(dup, ip)
	for j := len(dup) - 1; j >= 0; j-- {
		dup[j]++
		if dup[j] > 0 {
			break
		}
	}
	return dup
}
