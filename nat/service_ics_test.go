//+build windows

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
	"errors"
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

var _ NATService = &serviceICS{}

func mockedICS(powerShell func(cmd string) ([]byte, error)) *serviceICS {
	return &serviceICS{
		powerShell:      powerShell,
		ifaces:          make(map[string]RuleForwarding),
		setICSAddresses: mockICSConfig,
	}
}

func Test_emptyActionForSharing(t *testing.T) {
	sh := mockPowerShell{}
	ics := mockedICS(sh.exec)
	err := ics.applySharingConfig("", "eth0")
	assert.EqualError(t, err, "empty action provided")
}

func Test_emptyInterfaceNameForSharing(t *testing.T) {
	sh := mockPowerShell{}
	ics := mockedICS(sh.exec)
	err := ics.applySharingConfig(disableSharing, "")
	assert.EqualError(t, err, "empty interface name provided")
}

func Test_errorOnApplySharing(t *testing.T) {
	sh := mockPowerShell{err: errors.New("expected error")}
	ics := mockedICS(sh.exec)
	err := ics.applySharingConfig(disableSharing, "eth0")
	assert.EqualError(t, err, "expected error")
}

func Test_validApplySharing(t *testing.T) {
	sh := mockPowerShell{}
	ics := mockedICS(sh.exec)
	err := ics.applySharingConfig(disableSharing, "eth0")
	assert.NoError(t, err)
}

func Test_errorOnGetPublicInterfaceName(t *testing.T) {
	sh := mockPowerShell{err: errors.New("expected error")}
	ics := mockedICS(sh.exec)
	_, err := ics.getPublicInterfaceName()
	assert.EqualError(t, err, "failed to get interface from the default route: expected error")
}

func Test_parseErrorOnGetPublicInterfaceName(t *testing.T) {
	sh := mockPowerShell{}
	ics := mockedICS(sh.exec)
	_, err := ics.getPublicInterfaceName()
	assert.EqualError(t, err, "failed to parse interface ID: strconv.Atoi: parsing \"\": invalid syntax")
}

func Test_getPublicInterfaceName(t *testing.T) {
	sh := mockPowerShell{commands: map[string]mockShellResult{
		"Get-WmiObject -Class Win32_IP4RouteTable | where { $_.destination -eq '0.0.0.0' -and $_.mask -eq '0.0.0.0'} | foreach { $_.InterfaceIndex }": {[]byte("1234567890"), nil},
	}}
	ics := mockedICS(sh.exec)
	_, err := ics.getPublicInterfaceName()
	assert.EqualError(t, err, "interface not found")
}

func Test_errorGettingServiceStartTypeOnEnable(t *testing.T) {
	sh := mockPowerShell{commands: map[string]mockShellResult{
		"Get-Service RemoteAccess | foreach { $_.StartType }": {nil, errors.New("expected error")},
	}}

	ics := mockedICS(sh.exec)
	err := ics.Enable()
	assert.EqualError(t, err, "failed to Enable RemoteAccess service: failed to get RemoteAccess service startup type: expected error")
}

func Test_errorSettingServiceStartTypeOnEnable(t *testing.T) {
	sh := mockPowerShell{commands: map[string]mockShellResult{
		"Set-Service -Name RemoteAccess -StartupType automatic": {nil, errors.New("expected error")},
	}}

	ics := mockedICS(sh.exec)
	err := ics.Enable()
	assert.EqualError(t, err, "failed to Enable RemoteAccess service: failed to set RemoteAccess service startup type to automatic: expected error")
}

func Test_errorStartingServiceOnEnable(t *testing.T) {
	sh := mockPowerShell{commands: map[string]mockShellResult{
		"Start-Service -Name RemoteAccess": {nil, errors.New("expected error")},
	}}

	ics := mockedICS(sh.exec)
	err := ics.Enable()
	assert.EqualError(t, err, "failed to Enable RemoteAccess service: failed to start RemoteAccess service: expected error")
}

func Test_errorGettingPublicInterfaceOnEnable(t *testing.T) {
	sh := mockPowerShell{commands: map[string]mockShellResult{
		"Get-WmiObject -Class Win32_IP4RouteTable | where { $_.destination -eq '0.0.0.0' -and $_.mask -eq '0.0.0.0'} | foreach { $_.InterfaceIndex }": {nil, errors.New("expected error")},
	}}

	ics := mockedICS(sh.exec)
	err := ics.Enable()
	assert.EqualError(t, err, "failed to get public interface name: failed to get interface from the default route: expected error")
}

func Test_errorApplyingSharingConfigOnEnable(t *testing.T) {
	ifaces, _ := net.Interfaces()
	iface := ifaces[0]

	sh := mockPowerShell{commands: map[string]mockShellResult{
		"Get-WmiObject -Class Win32_IP4RouteTable | where { $_.destination -eq '0.0.0.0' -and $_.mask -eq '0.0.0.0'} | foreach { $_.InterfaceIndex }": {[]byte(strconv.Itoa(iface.Index)), nil},

		`regsvr32 /s hnetcfg.dll;
		$netShare = New-Object -ComObject HNetCfg.HNetShare;
		$c = $netShare.EnumEveryConnection |? { $netShare.NetConnectionProps.Invoke($_).Name -eq "` + iface.Name + `" };
		$config = $netShare.INetSharingConfigurationForINetConnection.Invoke($c);$config.EnableSharing(0)`: {nil, errors.New("expected error")},
	}}

	ics := mockedICS(sh.exec)
	err := ics.Enable()
	assert.EqualError(t, err, "failed to enable internet connection sharing: expected error")
}

func Test_Enable(t *testing.T) {
	ifaces, _ := net.Interfaces()
	iface := ifaces[0]

	sh := mockPowerShell{commands: map[string]mockShellResult{
		"Get-WmiObject -Class Win32_IP4RouteTable | where { $_.destination -eq '0.0.0.0' -and $_.mask -eq '0.0.0.0'} | foreach { $_.InterfaceIndex }": {[]byte(strconv.Itoa(iface.Index)), nil},
	}}

	ics := mockedICS(sh.exec)
	err := ics.Enable()
	assert.NoError(t, err)
}

func Test_AddDel(t *testing.T) {
	sh := mockPowerShell{commands: map[string]mockShellResult{
		`Get-WmiObject Win32_NetworkAdapter | Where-Object {$_.ServiceName -eq "tap0901"} | foreach { $_.NetConnectionID }`: {[]byte("myst0"), nil},
	}}

	ics := mockedICS(sh.exec)
	err := ics.Add(RuleForwarding{"127.0.0.1/24", "8.8.8.8"})
	assert.NoError(t, err)

	ifaceName, err := ics.getInternalInterfaceName()
	assert.NoError(t, err)

	_, ok := ics.ifaces[ifaceName]
	assert.Equal(t, true, ok)

	err = ics.Del(RuleForwarding{"127.0.0.1/24", "8.8.8.8"})
	assert.NoError(t, err)

	_, ok = ics.ifaces[ifaceName]
	assert.Equal(t, false, ok)
}

func Test_errorInterfaceOnAdd(t *testing.T) {
	sh := mockPowerShell{commands: map[string]mockShellResult{}}

	ics := mockedICS(sh.exec)
	err := ics.Add(RuleForwarding{"8.8.8.8/24", "8.8.8.8"})
	assert.EqualError(t, err, "failed to find suitable interface: interface not found")
}

func Test_errorInterfaceOnDel(t *testing.T) {
	sh := mockPowerShell{commands: map[string]mockShellResult{}}

	ics := mockedICS(sh.exec)
	err := ics.Del(RuleForwarding{"8.8.8.8/24", "8.8.8.8"})
	assert.EqualError(t, err, "failed to find suitable interface: interface not found")
}

func Test_errorRevertStartupTypeOnDisable(t *testing.T) {
	ifaces, _ := net.Interfaces()
	iface := ifaces[0]

	sh := mockPowerShell{commands: map[string]mockShellResult{
		"Get-WmiObject -Class Win32_IP4RouteTable | where { $_.destination -eq '0.0.0.0' -and $_.mask -eq '0.0.0.0'} | foreach { $_.InterfaceIndex }": {[]byte(strconv.Itoa(iface.Index)), nil},
		"Set-Service -Name RemoteAccess -StartupType testStatus": {nil, errors.New("expected error")},
	}}

	ics := mockedICS(sh.exec)
	ics.remoteAccessStatus = "testStatus"
	err := ics.Disable()

	assert.EqualError(t, err, "failed to revert RemoteAccess service startup type: expected error")
}

func Test_errorGettingPublicInterfaceOnDisable(t *testing.T) {
	sh := mockPowerShell{commands: map[string]mockShellResult{
		"Get-WmiObject -Class Win32_IP4RouteTable | where { $_.destination -eq '0.0.0.0' -and $_.mask -eq '0.0.0.0'} | foreach { $_.InterfaceIndex }": {nil, errors.New("expected error")},
	}}

	ics := mockedICS(sh.exec)
	err := ics.Disable()

	assert.EqualError(t, err, "failed to get public interface name: failed to get interface from the default route: expected error")
}

func Test_Disable(t *testing.T) {
	ifaces, _ := net.Interfaces()
	iface := ifaces[0]

	sh := mockPowerShell{commands: map[string]mockShellResult{
		"Get-WmiObject -Class Win32_IP4RouteTable | where { $_.destination -eq '0.0.0.0' -and $_.mask -eq '0.0.0.0'} | foreach { $_.InterfaceIndex }": {[]byte(strconv.Itoa(iface.Index)), nil},
	}}

	ics := mockedICS(sh.exec)
	err := ics.Disable()

	assert.NoError(t, err)
}

type mockShellResult struct {
	output []byte
	err    error
}

type mockPowerShell struct {
	err      error
	commands map[string]mockShellResult
}

func (sh *mockPowerShell) exec(cmd string) ([]byte, error) {
	if c, ok := sh.commands[cmd]; ok {
		return c.output, c.err
	}
	return nil, sh.err
}

func mockICSConfig(_map map[string]string) (map[string]string, error) {
	return nil
}
