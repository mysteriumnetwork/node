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
	"net/url"
	"strings"
	"sync"

	"github.com/mysteriumnetwork/node/firewall/iptables"
	"github.com/rs/zerolog/log"
)

const killswitchChain = "MYST_CONSUMER_KILL_SWITCH"

type refCount struct {
	count int
	f     func()
}

type outgoingFirewallIptables struct {
	lock             sync.Mutex
	trafficLockScope Scope
	referenceTracker map[string]refCount
}

// Setup tries to setup all changes made by setup and leave system in the state before setup.
func (obi *outgoingFirewallIptables) Setup() error {
	if err := obi.checkIptablesVersion(); err != nil {
		return err
	}
	if err := obi.cleanupStaleRules(); err != nil {
		return err
	}
	return obi.setupKillSwitchChain()
}

// Teardown tries to cleanup all changes made by setup and leave system in the state before setup.
func (obi *outgoingFirewallIptables) Teardown() {
	if err := obi.cleanupStaleRules(); err != nil {
		log.Warn().Err(err).Msg("Error cleaning up iptables rules, you might want to do it yourself")
	}
}

// BlockOutgoingTraffic effectively disallows any outgoing traffic from consumer node with specified scope.
func (obi *outgoingFirewallIptables) BlockOutgoingTraffic(scope Scope, outboundIP string) (OutgoingRuleRemove, error) {
	if obi.trafficLockScope == Global {
		// nothing can override global lock
		return func() {}, nil
	}
	obi.trafficLockScope = scope
	return obi.trackingReferenceCall("block-traffic", func() (OutgoingRuleRemove, error) {
		// Take custom chain into effect for packets in OUTPUT
		return iptables.AddRuleWithRemoval(
			iptables.AppendTo("OUTPUT").RuleSpec("-s", outboundIP, "-j", killswitchChain),
		)
	})
}

// AllowIPAccess adds exception to blocked traffic for specified URL (host part is usually taken).
func (obi *outgoingFirewallIptables) AllowIPAccess(ip string) (OutgoingRuleRemove, error) {
	return obi.trackingReferenceCall("allow:"+ip, func() (rule OutgoingRuleRemove, e error) {
		return iptables.AddRuleWithRemoval(
			iptables.InsertAt(killswitchChain, 1).RuleSpec("-d", ip, "-j", "ACCEPT"),
		)
	})
}

// AllowURLAccess adds URL based exception.
func (obi *outgoingFirewallIptables) AllowURLAccess(rawURLs ...string) (OutgoingRuleRemove, error) {
	var ruleRemovers []func()
	removeAll := func() {
		for _, ruleRemover := range ruleRemovers {
			ruleRemover()
		}
	}
	for _, rawURL := range rawURLs {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			removeAll()
			return nil, err
		}

		remover, err := obi.AllowIPAccess(parsed.Hostname())
		if err != nil {
			removeAll()
			return nil, err
		}
		ruleRemovers = append(ruleRemovers, remover)
	}
	return removeAll, nil
}

func (obi *outgoingFirewallIptables) checkIptablesVersion() error {
	output, err := iptables.Exec("--version")
	if err != nil {
		return err
	}
	for _, line := range output {
		log.Info().Msg("[version check] " + line)
	}
	return nil
}

func (obi *outgoingFirewallIptables) setupKillSwitchChain() error {
	// Add chain
	if _, err := iptables.Exec("-N", killswitchChain); err != nil {
		return err
	}
	// Append rule - by default all packets going to kill switch chain are rejected
	if _, err := iptables.Exec("-A", killswitchChain, "-m", "conntrack", "--ctstate", "NEW", "-j", "REJECT"); err != nil {
		return err
	}

	// Insert rule - TODO for now always allow outgoing DNS traffic, BUT it should be exposed as separate firewall call
	if _, err := iptables.Exec("-I", killswitchChain, "1", "-p", "udp", "--dport", "53", "-j", "ACCEPT"); err != nil {
		return err
	}
	// Insert rule - TCP DNS is not so popular - but for the sake of humanity, lets allow it too
	if _, err := iptables.Exec("-I", killswitchChain, "1", "-p", "tcp", "--dport", "53", "-j", "ACCEPT"); err != nil {
		return err
	}

	return nil
}

func (obi *outgoingFirewallIptables) cleanupStaleRules() error {
	// List rules
	rules, err := iptables.Exec("-S", "OUTPUT")
	if err != nil {
		return err
	}
	for _, rule := range rules {
		// detect if any references exist in OUTPUT chain like -j MYST_CONSUMER_KILL_SWITCH
		if strings.HasSuffix(rule, killswitchChain) {
			deleteRule := strings.Replace(rule, "-A", "-D", 1)
			deleteRuleArgs := strings.Split(deleteRule, " ")
			if _, err := iptables.Exec(deleteRuleArgs...); err != nil {
				return err
			}
		}
	}

	// List chain rules
	if _, err := iptables.Exec("-L", killswitchChain); err != nil {
		// error means no such chain - log error just in case and bail out
		log.Info().Err(err).Msg("[setup] Got error while listing kill switch chain rules. Probably nothing to worry about")
		return nil
	}

	// Remove chain rules
	if _, err := iptables.Exec("-F", killswitchChain); err != nil {
		return err
	}

	// Remove chain
	_, err = iptables.Exec("-X", killswitchChain)
	return err
}

func (obi *outgoingFirewallIptables) trackingReferenceCall(ref string, actualCall func() (OutgoingRuleRemove, error)) (OutgoingRuleRemove, error) {
	obi.lock.Lock()
	defer obi.lock.Unlock()

	refCount := obi.referenceTracker[ref]
	if refCount.count == 0 {
		removeRule, err := actualCall()
		if err != nil {
			return nil, err
		}
		refCount.f = removeRule

		refCount.count++
		obi.referenceTracker[ref] = refCount
	}

	return obi.decreaseRefCall(ref), nil
}

func (obi *outgoingFirewallIptables) decreaseRefCall(ref string) OutgoingRuleRemove {
	return func() {
		obi.lock.Lock()
		defer obi.lock.Unlock()

		refCount := obi.referenceTracker[ref]
		if refCount.count == 1 {
			refCount.f()

			refCount.count--
			obi.referenceTracker[ref] = refCount
		}
	}
}

var _ OutgoingTrafficFirewall = &outgoingFirewallIptables{}
