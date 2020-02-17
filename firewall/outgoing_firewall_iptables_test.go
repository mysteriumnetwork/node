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
	"testing"

	"github.com/mysteriumnetwork/node/firewall/iptables"
	"github.com/stretchr/testify/assert"
)

func Test_outgoingFirewallIptables_BlocksAllOutgoingTraffic(t *testing.T) {
	mockedExec := iptablesExecMock{
		mocks: map[string]iptablesExecResult{},
	}
	iptables.Exec = mockedExec.Exec

	fw := &outgoingFirewallIptables{
		referenceTracker: make(map[string]refCount),
	}

	removeRuleFunc, err := fw.BlockOutgoingTraffic("test-scope", "1.1.1.1")
	assert.NoError(t, err)
	assert.True(t, mockedExec.VerifyCalledWithArgs("-A", "OUTPUT", "-s", "1.1.1.1", "-j", killswitchChain))

	removeRuleFunc()
	assert.True(t, mockedExec.VerifyCalledWithArgs("-D", "OUTPUT", "-s", "1.1.1.1", "-j", killswitchChain))
}

func Test_outgoingFirewallIptables_SessionTrafficBlockIsNoopWhenGlobalBlockWasCalled(t *testing.T) {
	mockedExec := iptablesExecMock{
		mocks: map[string]iptablesExecResult{},
	}
	iptables.Exec = mockedExec.Exec

	fw := &outgoingFirewallIptables{
		referenceTracker: make(map[string]refCount),
	}

	removeGlobalBlock, err := fw.BlockOutgoingTraffic(Global, "1.1.1.1")
	assert.NoError(t, err)
	assert.Equal(t, 1, fw.referenceTracker["block-traffic"].count)
	assert.True(t, mockedExec.VerifyCalledWithArgs("-A", "OUTPUT", "-s", "1.1.1.1", "-j", killswitchChain))

	removeSessionRule, _ := fw.BlockOutgoingTraffic(Session, "1.1.1.1")
	assert.Equal(t, 1, fw.referenceTracker["block-traffic"].count)

	removeSessionRule()
	assert.Equal(t, 1, fw.referenceTracker["block-traffic"].count)

	removeGlobalBlock()
	assert.Equal(t, 0, fw.referenceTracker["block-traffic"].count)
}

func Test_outgoingFirewallIptables_AllowIPAccessIsAddedAndRemoved(t *testing.T) {
	fw := &outgoingFirewallIptables{
		referenceTracker: make(map[string]refCount),
	}

	removeRule, _ := fw.AllowIPAccess("test-ip")
	assert.Equal(t, 1, fw.referenceTracker["allow:test-ip"].count)
	removeRule()
	assert.Equal(t, 0, fw.referenceTracker["allow:test-ip"].count)
}

func Test_outgoingFirewallIptables_HostsFromMultipleURLsAreAllowed(t *testing.T) {
	fw := &outgoingFirewallIptables{
		referenceTracker: make(map[string]refCount),
	}

	removeRules, _ := fw.AllowURLAccess("http://url1", "my-schema://url2:500/ignoredpath?ignoredQuery=true")
	assert.Equal(t, 1, fw.referenceTracker["allow:url1"].count)
	assert.Equal(t, 1, fw.referenceTracker["allow:url2"].count)
	removeRules()
	assert.Equal(t, 0, fw.referenceTracker["allow:url1"].count)
	assert.Equal(t, 0, fw.referenceTracker["allow:url2"].count)
}

func Test_outgoingFirewallIptables_RuleIsRemovedOnlyAfterLastRemovalCall(t *testing.T) {
	fw := &outgoingFirewallIptables{
		referenceTracker: make(map[string]refCount),
	}

	//two independent allow requests for the same service
	removalRequest1, _ := fw.AllowIPAccess("service")
	removalRequest2, _ := fw.AllowIPAccess("service")
	//make sure allow ip was called once
	assert.Equal(t, 1, fw.referenceTracker["allow:service"].count)
	//first removal should have no effect
	removalRequest1()
	assert.Equal(t, 0, fw.referenceTracker["allow:service"].count)
	//second removal removes added rule
	removalRequest2()
	assert.Equal(t, 0, fw.referenceTracker["allow:service"].count)
}

func Test_outgoingFirewallIptables_SetupIsSuccessful(t *testing.T) {
	mockedExec := iptablesExecMock{
		mocks: map[string]iptablesExecResult{
			"--version": {
				output: []string{"iptables v1.6.0"},
			},
			"-S OUTPUT": {
				output: []string{
					"-P OUTPUT ACCEPT",
				},
			},
		},
	}
	iptables.Exec = mockedExec.Exec

	fw := &outgoingFirewallIptables{
		referenceTracker: make(map[string]refCount),
	}
	assert.NoError(t, fw.Setup())
	assert.True(t, mockedExec.VerifyCalledWithArgs("-N", killswitchChain))
	assert.True(t, mockedExec.VerifyCalledWithArgs("-A", killswitchChain, "-m", "conntrack", "--ctstate", "NEW", "-j", "REJECT"))
}

func Test_outgoingFirewallIptables_SetupIsSucessfulIfPreviousCleanupFailed(t *testing.T) {
	mockedExec := iptablesExecMock{
		mocks: map[string]iptablesExecResult{
			"--version": {
				output: []string{"iptables v1.6.0"},
			},
			"-S OUTPUT": {
				output: []string{
					"-P OUTPUT ACCEPT",
					// leftover - kill switch is still enabled
					"-A OUTPUT -s 5.5.5.5 -j MYST_CONSUMER_KILL_SWITCH",
				},
			},
			// kill switch chain still exists
			"-S MYST_CONSUMER_KILL_SWITCH": {
				output: []string{
					// with some allowed ips
					"-A MYST_CONSUMER_KILL_SWITCH -d 2.2.2.2 -j ACCEPT",
					"-A MYST_CONSUMER_KILL_SWITCH -j REJECT",
				},
			},
		},
	}
	iptables.Exec = mockedExec.Exec

	fw := &outgoingFirewallIptables{
		referenceTracker: make(map[string]refCount),
	}
	assert.NoError(t, fw.Setup())
	assert.True(t, mockedExec.VerifyCalledWithArgs("-D", "OUTPUT", "-s", "5.5.5.5", "-j", killswitchChain))
	assert.True(t, mockedExec.VerifyCalledWithArgs("-F", killswitchChain))
	assert.True(t, mockedExec.VerifyCalledWithArgs("-X", killswitchChain))
	assert.True(t, mockedExec.VerifyCalledWithArgs("-N", killswitchChain))
	assert.True(t, mockedExec.VerifyCalledWithArgs("-A", killswitchChain, "-m", "conntrack", "--ctstate", "NEW", "-j", "REJECT"))

}

func Test_outgoingFirewallIptables_ResetIsSuccessful(t *testing.T) {
	mockedExec := iptablesExecMock{
		mocks: map[string]iptablesExecResult{
			"-S OUTPUT": {
				output: []string{
					"-P OUTPUT ACCEPT",
					// kill switch is enabled
					"-A OUTPUT -s 1.1.1.1 -j MYST_CONSUMER_KILL_SWITCH",
				},
			},
			"-S MYST_CONSUMER_KILL_SWITCH": {
				output: []string{
					//first allowed address
					"-A MYST_CONSUMER_KILL_SWITCH -d 2.2.2.2 -j ACCEPT",
					//second allowed address
					"-A MYST_CONSUMER_KILL_SWITCH -d 3.3.3.3 -j ACCEPT",
					//drop everything else
					"-A MYST_CONSUMER_KILL_SWITCH -j REJECT",
				},
			},
		},
	}
	iptables.Exec = mockedExec.Exec

	fw := &outgoingFirewallIptables{
		referenceTracker: make(map[string]refCount),
	}
	fw.Teardown()
	assert.True(t, mockedExec.VerifyCalledWithArgs("-D", "OUTPUT", "-s", "1.1.1.1", "-j", killswitchChain))
	assert.True(t, mockedExec.VerifyCalledWithArgs("-F", killswitchChain))
	assert.True(t, mockedExec.VerifyCalledWithArgs("-X", killswitchChain))
}

func Test_outgoingFirewallIptables_AddsAllowedIP(t *testing.T) {
	mockedExec := iptablesExecMock{
		mocks: map[string]iptablesExecResult{},
	}
	iptables.Exec = mockedExec.Exec

	fw := &outgoingFirewallIptables{
		referenceTracker: make(map[string]refCount),
	}

	removeRuleFunc, err := fw.AllowIPAccess("2.2.2.2")
	assert.NoError(t, err)
	assert.True(t, mockedExec.VerifyCalledWithArgs("-I", killswitchChain, "1", "-d", "2.2.2.2", "-j", "ACCEPT"))

	removeRuleFunc()
	assert.True(t, mockedExec.VerifyCalledWithArgs("-D", killswitchChain, "-d", "2.2.2.2", "-j", "ACCEPT"))

}
