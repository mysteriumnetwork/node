/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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
	"os/exec"

	"github.com/pkg/errors"
)

func NewService() NATService {
	return &NATWindows{}
}

type NATWindows struct {
	remoteAccessStatus []byte
}

func (nat *NATWindows) Enable() error {
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

	// Enable internet connection sharing for the "Ethernet" adapter as a public interface.
	// TODO detect public interface automatically.
	_, err := powerShell(`regsvr32 /s hnetcfg.dll;
		$netShare = New-Object -ComObject HNetCfg.HNetShare;
		$c = $netShare.EnumEveryConnection |? { $netShare.NetConnectionProps.Invoke($_).Name -eq "Ethernet" };
		$config = $netShare.INetSharingConfigurationForINetConnection.Invoke($c);
		$config.EnableSharing(0)`)

	return err
}

func (nat *NATWindows) Add(rule RuleForwarding) error {
	// Enable internet connection sharing for the local interface.
	// TODO detect interace name from the rule.
	ifaceName = "myst1"

	_, err := powerShell(`regsvr32 /s hnetcfg.dll;
		$netShare = New-Object -ComObject HNetCfg.HNetShare;
		$c = $netShare.EnumEveryConnection |? { $netShare.NetConnectionProps.Invoke($_).Name -eq "` + ifaceName + `" };
		$config = $netShare.INetSharingConfigurationForINetConnection.Invoke($c);
		$config.EnableSharing(1)`)

	return err
}

func (nat *NATWindows) Del(rule RuleForwarding) error {
	// Disable internet connection sharing for the local interface.
	// TODO detect interace name from the rule.
	ifaceName = "myst1"

	_, err := powerShell(`regsvr32 /s hnetcfg.dll;
		$netShare = New-Object -ComObject HNetCfg.HNetShare;
		$c = $netShare.EnumEveryConnection |? { $netShare.NetConnectionProps.Invoke($_).Name -eq "` + ifaceName + `" };
		$config = $netShare.INetSharingConfigurationForINetConnection.Invoke($c);
		$config.DisableSharing()`)

	return err
}

func (nat *NATWindows) Disable() error {
	_, err := powerShell("Set-Service -Name RemoteAccess -StartupType " + nat.remoteAccessStatus)
	if err != nil {
		return
	}

	// Disable internet connection sharing for the "Ethernet" adapter as a public interface.
	// TODO detect public interface automatically.
	_, err := powerShell(`regsvr32 /s hnetcfg.dll;
		$netShare = New-Object -ComObject HNetCfg.HNetShare;
		$c = $netShare.EnumEveryConnection |? { $netShare.NetConnectionProps.Invoke($_).Name -eq "Ethernet" };
		$config = $netShare.INetSharingConfigurationForINetConnection.Invoke($c);
		$config.DisableSharing()`)

	return err
}

func powerShell(cmd string) ([]byte, error) {
	out, err := exec.Command("powershell", "-Command", cmd).CombinedOutput()
	return out, errors.Wrap(err, string(out))
}
