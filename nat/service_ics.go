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
	propsForPublic       = `$props.IsIcsPublic = 1;$props.IsIcsPrivate = 0;`
	propsForPrivate      = `$props.IsIcsPublic = 0;$props.IsIcsPrivate = 1;`
	propsForDisable      = `$props.IsIcsPublic = 0;$props.IsIcsPrivate = 0;`
)

func getSharingScript(ifaceName, props, action string) string {
	return `regsvr32 /s hnetcfg.dll;
		$netShare = New-Object -ComObject HNetCfg.HNetShare;
		$c = $netShare.EnumEveryConnection |? { $netshare.NetConnectionProps.Invoke($_).Name -eq "` + ifaceName + `" };
		$config = $netShare.INetSharingConfigurationForINetConnection.Invoke($c);
		Try {
			` + action + `
		} Catch {
			$guid = $netShare.NetConnectionProps.Invoke($c).Guid;
			$props = Get-WmiObject -Class HNet_ConnectionProperties -Namespace "ROOT\microsoft\homenet" -Filter "__PATH like '%$guid%'";` + props + `$props.Put();
			` + action + `
		}`
}

func getPublicSharingScript(ifaceName string) string {
	return getSharingScript(ifaceName, propsForPublic, enablePublicSharing)
}

func getPrivateSharingScript(ifaceName string) string {
	return getSharingScript(ifaceName, propsForPrivate, enablePrivateSharing)
}

func getDisableSharingScript(ifaceName string) string {
	return getSharingScript(ifaceName, propsForDisable, disableSharing)
}

type serviceICS struct {
	mu                 sync.Mutex
	ifaces             map[string]RuleForwarding // list in internal interfaces with enabled internet connection sharing
	remoteAccessStatus string
	powerShell         func(cmd string) ([]byte, error)
	setICSAddresses    func(config map[string]string) (map[string]string, error)
	oldICSConfig       map[string]string
}

func (ics *serviceICS) disableICSAllInterfaces() error {
	// Filter out invalid media (where NETCON_MEDIATYPE = NCM_NONE) because it is not an instance of INetConnection
	// and will throw a cast error when attempting to `INetSharingConfigurationForINetConnection($conn)`
	// Enum: https://docs.microsoft.com/en-us/windows/desktop/api/netcon/ne-netcon-tagnetcon_mediatype
	_, err := ics.powerShell(`regsvr32 /s hnetcfg.dll;
		$mgr = New-Object -ComObject HnetCfg.HNetShare;
		filter IsValidMediaType { if ($mgr.NetConnectionProps($_).MediaType -gt 0) { $_ } };
		$connections = $mgr.EnumEveryConnection | IsValidMediaType;
		foreach ($conn in $connections) {
			$mgr.INetSharingConfigurationForINetConnection($conn).DisableSharing()
		};
	`)
	return err
}

// Enable enables internet connection sharing for the public interface.
func (ics *serviceICS) Enable() error {
	// We have to clean up ICS configuration for all interfaces to apply our configuration.
	// It is possible to have ICS configured only for single pair of interfaces.
	if err := ics.disableICSAllInterfaces(); err != nil {
		return errors.Wrap(err, "failed to cleanup ICS before Enabling")
	}

	if err := ics.enableRemoteAccessService(); err != nil {
		return errors.Wrap(err, "failed to Enable RemoteAccess service")
	}

	ifaceName, err := ics.getPublicInterfaceName()
	if err != nil {
		return errors.Wrap(err, "failed to get public interface name")
	}

	_, err = ics.powerShell(getPublicSharingScript(ifaceName))
	return errors.Wrap(err, "failed to enable internet connection sharing")
}

func (ics *serviceICS) enableRemoteAccessService() error {
	status, err := ics.powerShell(`(Get-WmiObject Win32_Service -filter "Name='RemoteAccess'").StartMode`)
	if err != nil {
		return errors.Wrap(err, "failed to get RemoteAccess service startup type")
	}

	statusStringified := strings.ToLower(strings.TrimSpace(string(status)))
	if statusStringified == "auto" {
		ics.remoteAccessStatus = "automatic"
	} else {
		ics.remoteAccessStatus = statusStringified
	}

	if _, err := ics.powerShell("Set-Service -Name RemoteAccess -StartupType automatic"); err != nil {
		return errors.Wrap(err, "failed to set RemoteAccess service startup type to automatic")
	}

	_, err = ics.powerShell("Start-Service -Name RemoteAccess")
	return errors.Wrap(err, "failed to start RemoteAccess service")
}

// Add enables internet connection sharing for the local interface.
func (ics *serviceICS) Add(rule RuleForwarding) error {
	_, ipnet, err := net.ParseCIDR(rule.SourceSubnet)
	if err != nil {
		log.Warnf("%s Failed to parse IP-address: %s", natLogPrefix, rule.SourceSubnet)
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

	_, err = ics.powerShell(getPrivateSharingScript(ifaceName))
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
	ifaceName, err := ics.getInternalInterfaceName()
	if err != nil {
		return errors.Wrap(err, "failed to find suitable interface")
	}

	_, err = ics.powerShell(getDisableSharingScript(ifaceName))
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

	_, err = ics.powerShell(getDisableSharingScript(ifaceName))
	if err != nil {
		err = errors.Wrap(err, "failed to disable internet connection sharing")
		log.Errorf("%s %v", natLogPrefix, err)
		if resErr == nil {
			resErr = err
		}
	}

	if err := ics.disableICSAllInterfaces(); err != nil {
		return errors.Wrap(err, "failed to cleanup ICS before Enabling")
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
