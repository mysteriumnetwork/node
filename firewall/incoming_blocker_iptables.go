/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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
	"strings"

	"github.com/mysteriumnetwork/node/firewall/ipset"
	"github.com/mysteriumnetwork/node/firewall/iptables"
	"github.com/rs/zerolog/log"
)

const (
	dnsFirewallChain = "PROVIDER_DNS_FIREWALL"
	dnsFirewallIpset = "myst-provider-dst-whitelist"
)

// NewIncomingTrafficBlockerIptables creates instance iptables based traffic blocker
func NewIncomingTrafficBlockerIptables() IncomingTrafficBlocker {
	return &incomingBlockerIptables{}
}

// incomingBlockerIptables allows incoming traffic blocking in IP granularity
type incomingBlockerIptables struct{}

func (ibi *incomingBlockerIptables) Setup() error {
	if err := ibi.checkIpsetVersion(); err != nil {
		return err
	}

	// Clean up setups from previous runs, just in case
	if err := ibi.cleanupStaleRules(); err != nil {
		return err
	}
	ipset.Exec(ipset.OpDelete(dnsFirewallIpset))

	op := ipset.OpCreate(dnsFirewallIpset, ipset.SetTypeHashIP, nil, 0)
	if _, err := ipset.Exec(op); err != nil {
		return err
	}
	return ibi.setupDNSFirewallChain()
}

func (ibi *incomingBlockerIptables) Teardown() {
	if err := ibi.cleanupStaleRules(); err != nil {
		log.Warn().Err(err).Msg("Error cleaning up iptables rules, you might want to do it yourself")
	}
	if errOutput, err := ipset.Exec(ipset.OpDelete(dnsFirewallIpset)); err != nil {
		log.Warn().Err(err).Msgf("Error deleting ipset table. %s", strings.Join(errOutput, ""))
	}
}

func (ibi *incomingBlockerIptables) BlockIncomingTraffic(network net.IPNet) (IncomingRuleRemove, error) {
	remover, err := iptables.AddRuleWithRemoval(
		iptables.AppendTo("FORWARD").RuleSpec("-s", network.String(), "-j", dnsFirewallChain),
	)
	if err != nil {
		return nil, err
	}
	return func() error {
		remover()
		return nil
	}, nil
}

func (ibi *incomingBlockerIptables) AllowIPAccess(ip net.IP) (IncomingRuleRemove, error) {
	if _, err := ipset.Exec(ipset.OpIPAdd(dnsFirewallIpset, ip)); err != nil {
		return nil, err
	}
	return func() error {
		_, err := ipset.Exec(ipset.OpIPRemove(dnsFirewallIpset, ip))
		return err
	}, nil
}

func (ibi *incomingBlockerIptables) checkIpsetVersion() error {
	output, err := ipset.Exec(ipset.OpVersion())
	if err != nil {
		return err
	}
	for _, line := range output {
		log.Info().Msg("[version check] " + line)
	}
	return nil
}

func (ibi *incomingBlockerIptables) setupDNSFirewallChain() error {
	// Add chain
	if _, err := iptables.Exec("-N", dnsFirewallChain); err != nil {
		return err
	}

	// Append rule - packets going to DNS firewall with these destination IPs are whitelisted
	if _, err := iptables.Exec("-A", dnsFirewallChain, "-m", "set", "--match-set", dnsFirewallIpset, "dst", "-j", "ACCEPT"); err != nil {
		return err
	}

	// Append rule - by default all packets going to DNS firewall chain are rejected
	if _, err := iptables.Exec("-A", dnsFirewallChain, "-j", "REJECT"); err != nil {
		return err
	}

	return nil
}

func (ibi *incomingBlockerIptables) cleanupStaleRules() error {
	// List rules
	rules, err := iptables.Exec("-S", "FORWARD")
	if err != nil {
		return err
	}
	for _, rule := range rules {
		// detect if any references exist in FORWARD chain like -j PROVIDER_DNS_FIREWALL
		if strings.HasSuffix(rule, dnsFirewallChain) {
			deleteRule := strings.Replace(rule, "-A", "-D", 1)
			deleteRuleArgs := strings.Split(deleteRule, " ")
			if _, err := iptables.Exec(deleteRuleArgs...); err != nil {
				return err
			}
		}
	}

	// List chain rules
	if _, err := iptables.Exec("-L", dnsFirewallChain); err != nil {
		// error means no such chain - log error just in case and bail out
		log.Info().Err(err).Msg("[setup] Got error while listing kill switch chain rules. Probably nothing to worry about")
		return nil
	}

	// Remove chain rules
	if _, err := iptables.Exec("-F", dnsFirewallChain); err != nil {
		return err
	}

	// Remove chain
	_, err = iptables.Exec("-X", dnsFirewallChain)
	return err
}

var _ IncomingTrafficBlocker = &incomingBlockerIptables{}
