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

package firewall

import (
	"net"
	"testing"

	"github.com/mysteriumnetwork/node/firewall/ipset"
	"github.com/mysteriumnetwork/node/firewall/iptables"
	"github.com/stretchr/testify/assert"
)

func Test_iptablesDNSFirewall_Setup(t *testing.T) {
	mockedIpset := ipsetExecMock{
		mocks: map[string]ipsetExecResult{
			"--version": {
				output: []string{
					"ipset v7.2, protocol version: 7",
					"Warning: Kernel support protocol versions 6-6 while userspace supports protocol versions 6-7",
				},
			},
			"-S FORWARD": {
				output: []string{"-P FORWARD ACCEPT"},
			},
		},
	}
	ipset.Exec = mockedIpset.Exec

	mockedIptables := iptablesExecMock{
		mocks: map[string]iptablesExecResult{},
	}
	iptables.Exec = mockedIptables.Exec

	firewall := &incomingBlockerIptables{}
	err := firewall.Setup()
	assert.NoError(t, err)
	assert.True(t, mockedIpset.VerifyCalledWithArgs("version"))
	assert.True(t, mockedIpset.VerifyCalledWithArgs("create myst-provider-dst-whitelist hash:ip"))
	assert.True(t, mockedIptables.VerifyCalledWithArgs("-N MYST_PROVIDER_FIREWALL"))
	assert.True(t, mockedIptables.VerifyCalledWithArgs("-A MYST_PROVIDER_FIREWALL -m set --match-set myst-provider-dst-whitelist dst -j ACCEPT"))
	assert.True(t, mockedIptables.VerifyCalledWithArgs("-A MYST_PROVIDER_FIREWALL -j REJECT"))
}

func Test_iptablesDNSFirewall_Teardown(t *testing.T) {
	mockedIpset := ipsetExecMock{
		mocks: map[string]ipsetExecResult{},
	}
	ipset.Exec = mockedIpset.Exec

	mockedIptables := iptablesExecMock{
		mocks: map[string]iptablesExecResult{
			"-S FORWARD": {
				output: []string{
					"-P FORWARD ACCEPT",
				},
			},
		},
	}
	iptables.Exec = mockedIptables.Exec

	firewall := &incomingBlockerIptables{}
	firewall.Teardown()
	assert.True(t, mockedIpset.VerifyCalledWithArgs("destroy myst-provider-dst-whitelist"))
	assert.True(t, mockedIptables.VerifyCalledWithArgs("-F MYST_PROVIDER_FIREWALL"))
	assert.True(t, mockedIptables.VerifyCalledWithArgs("-X MYST_PROVIDER_FIREWALL"))
}

func Test_iptablesDNSFirewall_TeardownIfPreviousCleanupFailed(t *testing.T) {
	mockedIpset := ipsetExecMock{
		mocks: map[string]ipsetExecResult{},
	}
	ipset.Exec = mockedIpset.Exec

	mockedIptables := iptablesExecMock{
		mocks: map[string]iptablesExecResult{
			"-S FORWARD": {
				output: []string{
					"-P FORWARD ACCEPT",
					// leftover - DNS direwall is still enabled
					"-A FORWARD -s 10.8.0.1/24 -j MYST_PROVIDER_FIREWALL",
				},
			},
			// DNS firewall chain still exists
			"-S MYST_PROVIDER_FIREWALL": {
				output: []string{
					// with some allowed ips
					"-N MYST_PROVIDER_FIREWALL",
					"-A MYST_PROVIDER_FIREWALL -m set --match-set myst-provider-dst-whitelist dst -j ACCEPT",
					"-A MYST_PROVIDER_FIREWALL -j REJECT --reject-with icmp-port-unreachable",
				},
			},
		},
	}
	iptables.Exec = mockedIptables.Exec

	firewall := &incomingBlockerIptables{}
	firewall.Teardown()
	assert.True(t, mockedIpset.VerifyCalledWithArgs("destroy myst-provider-dst-whitelist"))
	assert.True(t, mockedIptables.VerifyCalledWithArgs("-D FORWARD -s 10.8.0.1/24 -j MYST_PROVIDER_FIREWALL"))
	assert.True(t, mockedIptables.VerifyCalledWithArgs("-F MYST_PROVIDER_FIREWALL"))
	assert.True(t, mockedIptables.VerifyCalledWithArgs("-X MYST_PROVIDER_FIREWALL"))
}

func Test_iptablesDNSFirewall_BlockIncomingTraffic(t *testing.T) {
	mockedIptables := iptablesExecMock{
		mocks: map[string]iptablesExecResult{},
	}
	iptables.Exec = mockedIptables.Exec

	firewall := &incomingBlockerIptables{}

	_, network, _ := net.ParseCIDR("10.8.0.1/24")
	removeRule, err := firewall.BlockIncomingTraffic(*network)
	assert.NoError(t, err)
	assert.True(t, mockedIptables.VerifyCalledWithArgs("-A FORWARD -s 10.8.0.0/24 -j MYST_PROVIDER_FIREWALL"))

	removeRule()
	assert.True(t, mockedIptables.VerifyCalledWithArgs("-D FORWARD -s 10.8.0.0/24 -j MYST_PROVIDER_FIREWALL"))
}

func Test_iptablesDNSFirewall_AllowIPAccess(t *testing.T) {
	mockedIpset := ipsetExecMock{
		mocks: map[string]ipsetExecResult{},
	}
	ipset.Exec = mockedIpset.Exec

	firewall := &incomingBlockerIptables{}

	removeRule, err := firewall.AllowIPAccess(net.IP{1, 2, 3, 4})
	assert.NoError(t, err)
	assert.True(t, mockedIpset.VerifyCalledWithArgs("add myst-provider-dst-whitelist 1.2.3.4 --exist"))

	err = removeRule()
	assert.NoError(t, err)
	assert.True(t, mockedIpset.VerifyCalledWithArgs("del myst-provider-dst-whitelist 1.2.3.4"))
}
