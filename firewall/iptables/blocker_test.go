package iptables

import (
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlockerSetupIsSuccesful(t *testing.T) {
	mockedExec := &mockedCmdExec{
		mocks: map[string]cmdExecResult{
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
	iptablesExec = mockedExec.IptablesExec

	blocker := NewBlocker("1.1.1.1")
	assert.NoError(t, blocker.Setup())
	assert.True(t, mockedExec.VerifyCalledWithArgs(addChain, killswitchChain))
	assert.True(t, mockedExec.VerifyCalledWithArgs(addRule, killswitchChain, jumpTo, reject))
}

func TestBlockerSetupIsSucessfulIfPreviousCleanupFailed(t *testing.T) {
	mockedExec := &mockedCmdExec{
		mocks: map[string]cmdExecResult{
			"--version": {
				output: []string{"iptables v1.6.0"},
			},
			"-S OUTPUT": {
				output: []string{
					"-P OUTPUT ACCEPT",
					// leftover - kill switch is still enabled
					"-A OUTPUT -s 5.5.5.5 -j CONSUMER_KILL_SWITCH",
				},
			},
			// kill switch chain still exists
			"-S CONSUMER_KILL_SWITCH": {
				output: []string{
					// with some allowed ips
					"-A CONSUMER_KILL_SWITCH -d 2.2.2.2 -j ACCEPT",
					"-A CONSUMER_KILL_SWITCH -j REJECT",
				},
			},
		},
	}
	iptablesExec = mockedExec.IptablesExec

	blocker := NewBlocker("1.1.1.1")
	assert.NoError(t, blocker.Setup())
	assert.True(t, mockedExec.VerifyCalledWithArgs(removeRule, outputChain, sourceIP, "5.5.5.5", jumpTo, killswitchChain))
	assert.True(t, mockedExec.VerifyCalledWithArgs(removeChainRules, killswitchChain))
	assert.True(t, mockedExec.VerifyCalledWithArgs(removeChain, killswitchChain))
	assert.True(t, mockedExec.VerifyCalledWithArgs(addChain, killswitchChain))
	assert.True(t, mockedExec.VerifyCalledWithArgs(addRule, killswitchChain, jumpTo, reject))

}

func TestBlockerResetIsSuccessful(t *testing.T) {
	mockedExec := &mockedCmdExec{
		mocks: map[string]cmdExecResult{
			"-S OUTPUT": {
				output: []string{
					"-P OUTPUT ACCEPT",
					// kill switch is enabled
					"-A OUTPUT -s 1.1.1.1 -j CONSUMER_KILL_SWITCH",
				},
			},
			"-S CONSUMER_KILL_SWITCH": {
				output: []string{
					//first allowed address
					"-A CONSUMER_KILL_SWITCH -d 2.2.2.2 -j ACCEPT",
					//second allowed address
					"-A CONSUMER_KILL_SWITCH -d 3.3.3.3 -j ACCEPT",
					//drop everything else
					"-A CONSUMER_KILL_SWITCH -j REJECT",
				},
			},
		},
	}
	iptablesExec = mockedExec.IptablesExec

	blocker := NewBlocker("1.1.1.1")
	blocker.Reset()

	assert.True(t, mockedExec.VerifyCalledWithArgs(removeRule, outputChain, sourceIP, "1.1.1.1", jumpTo, killswitchChain))
	assert.True(t, mockedExec.VerifyCalledWithArgs(removeChainRules, killswitchChain))
	assert.True(t, mockedExec.VerifyCalledWithArgs(removeChain, killswitchChain))
}

func TestBlockerBlocksAllOutgoingTraffic(t *testing.T) {
	mockedExec := &mockedCmdExec{
		mocks: map[string]cmdExecResult{},
	}
	iptablesExec = mockedExec.IptablesExec

	blocker := NewBlocker("1.1.1.1")

	removeRuleFunc, err := blocker.BlockOutgoingTraffic()
	assert.NoError(t, err)
	assert.True(t, mockedExec.VerifyCalledWithArgs(addRule, outputChain, sourceIP, "1.1.1.1", jumpTo, killswitchChain))

	removeRuleFunc()
	assert.True(t, mockedExec.VerifyCalledWithArgs(removeRule, outputChain, sourceIP, "1.1.1.1", jumpTo, killswitchChain))
}

func TestBlockerAddsAllowedIP(t *testing.T) {
	mockedExec := &mockedCmdExec{
		mocks: map[string]cmdExecResult{},
	}
	iptablesExec = mockedExec.IptablesExec

	blocker := NewBlocker("1.1.1.1")

	removeRuleFunc, err := blocker.AllowIPAccess("2.2.2.2")
	assert.NoError(t, err)
	assert.True(t, mockedExec.VerifyCalledWithArgs(addRule, killswitchChain, destinationIP, "2.2.2.2", jumpTo, accept))

	removeRuleFunc()
	assert.True(t, mockedExec.VerifyCalledWithArgs(removeRule, killswitchChain, destinationIP, "2.2.2.2", jumpTo, accept))

}

type cmdExecResult struct {
	called bool
	output []string
	err    error
}

type mockedCmdExec struct {
	mocks map[string]cmdExecResult
}

func (mce *mockedCmdExec) IptablesExec(args ...string) ([]string, error) {
	key := argsToKey(args...)
	res := mce.mocks[key]
	res.called = true
	mce.mocks[key] = res
	return res.output, res.err
}

func (mce *mockedCmdExec) VerifyCalledWithArgs(args ...string) bool {
	key := argsToKey(args...)
	return mce.mocks[key].called
}

func argsToKey(args ...string) string {
	return strings.Join(args, " ")
}

func TestIPResolve(t *testing.T) {
	ips, err := net.LookupHost("216.58.209.3")
	assert.NoError(t, err)
	fmt.Println(ips)
}
