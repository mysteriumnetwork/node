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

package iptables

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/pkg/errors"
)

const (
	logPrefix = "[iptables] "

	outputChain     = "OUTPUT"
	killswitchChain = "CONSUMER_KILL_SWITCH"

	addChain         = "-N"
	appendRule       = "-A"
	insertRule       = "-I"
	listRules        = "-S"
	removeRule       = "-D"
	removeChainRules = "-F"
	removeChain      = "-X"

	jumpTo        = "-j"
	sourceIP      = "-s"
	destinationIP = "-d"
	module        = "-m"

	protocol        = "-p"
	tcp             = "tcp"
	udp             = "udp"
	destinationPort = "--dport"

	conntrack  = "conntrack"
	ctState    = "--ctstate"
	ctStateNew = "NEW"

	reject = "REJECT"
	accept = "ACCEPT"

	version = "--version"
)

var iptablesExec = func(args ...string) ([]string, error) {
	args = append([]string{"/sbin/iptables"}, args...)
	log.Trace(logPrefix, "[cmd] ", args)
	output, err := exec.Command("sudo", args...).CombinedOutput()
	if err != nil {
		log.Trace(logPrefix, "[cmd error] ", err, " ", args, " ", string(output))
		return nil, errors.Wrap(err, "iptables cmd error")
	}
	outputScanner := bufio.NewScanner(bytes.NewBuffer(output))
	var lines []string
	for outputScanner.Scan() {
		lines = append(lines, outputScanner.Text())
	}
	return lines, outputScanner.Err()
}

func checkVersion() error {
	output, err := iptablesExec(version)
	if err != nil {
		return err
	}
	for _, line := range output {
		log.Info(logPrefix, "[version check] ", line)
	}
	return nil
}

func cleanupStaleRules() error {
	rules, err := iptablesExec(listRules, outputChain)
	if err != nil {
		return err
	}
	for _, rule := range rules {
		//detect if any references exist in OUTPUT chain like -j CONSUMER_KILL_SWITCH
		if strings.HasSuffix(rule, killswitchChain) {
			deleteRule := strings.Replace(rule, appendRule, removeRule, 1)
			deleteRuleArgs := strings.Split(deleteRule, " ")
			if _, err := iptablesExec(deleteRuleArgs...); err != nil {
				return err
			}
		}
	}

	if _, err := iptablesExec(listRules, killswitchChain); err != nil {
		//error means no such chain - log error just in case and bail out
		log.Info(logPrefix, "[setup] Got error while listing kill switch chain rules. Probably nothing to worry about. Err: ", err)
		return nil
	}

	if _, err := iptablesExec(removeChainRules, killswitchChain); err != nil {
		return err
	}

	_, err = iptablesExec(removeChain, killswitchChain)
	return err
}

func setupKillSwitchChain() error {
	if _, err := iptablesExec(addChain, killswitchChain); err != nil {
		return err
	}
	// by default all packets going to kill switch chain are rejected
	if _, err := iptablesExec(appendRule, killswitchChain, module, conntrack, ctState, ctStateNew, jumpTo, reject); err != nil {
		return err
	}

	// TODO for now always allow outgoing DNS traffic, BUT it should be exposed as separate firewall call
	if _, err := iptablesExec(insertRule, killswitchChain, "1", protocol, udp, destinationPort, "53", jumpTo, accept); err != nil {
		return err
	}
	// TCP DNS is not so popular - but for the sake of humanity, lets allow it too
	if _, err := iptablesExec(insertRule, killswitchChain, "1", protocol, tcp, destinationPort, "53", jumpTo, accept); err != nil {
		return err
	}

	return nil
}

// Iptables represent Iptables based implementation of firewall Vendor interface
type Iptables struct {
	outboundIP string
}

// New initializes and returns Iptables with defined outboundIP
func New(outboundIP string) *Iptables {
	return &Iptables{
		outboundIP: outboundIP,
	}
}

// BlockOutgoingTraffic starts blocking outgoing traffic and returns function to remove the block
func (b Iptables) BlockOutgoingTraffic() (firewall.RemoveRule, error) {
	return addRuleWithRemoval(appendTo(outputChain).ruleSpec(sourceIP, b.outboundIP, jumpTo, killswitchChain))
}

// Setup prepares Iptables default rules and chains
func (b Iptables) Setup() error {
	if err := checkVersion(); err != nil {
		return err
	}

	if err := cleanupStaleRules(); err != nil {
		return err
	}

	return setupKillSwitchChain()
}

// Reset tries to cleanup all changes made by setup and leave system in the state before setup
func (Iptables) Reset() {
	if err := cleanupStaleRules(); err != nil {
		_ = log.Warn(logPrefix, "Error cleaning up iptables rules, you might want to do it yourself: ", err)
	}
}

func addRuleWithRemoval(chain chainInfo) (firewall.RemoveRule, error) {
	if _, err := iptablesExec(chain.applyArgs()...); err != nil {
		return nil, err
	}
	return func() {
		_, err := iptablesExec(chain.removeArgs()...)
		if err != nil {
			_ = log.Warn(logPrefix, "Error executing rule: ", chain.removeArgs(), " you might wanna do it yourself. Error was: ", err)
		}
	}, nil
}

// AllowIPAccess add ip to exceptions of blocked traffic and return function to remove exception
func (Iptables) AllowIPAccess(ip string) (firewall.RemoveRule, error) {
	return addRuleWithRemoval(insertAt(killswitchChain, 1).ruleSpec(destinationIP, ip, jumpTo, accept))
}

var _ firewall.Vendor = (*Iptables)(nil)
