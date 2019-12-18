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
	"strconv"
	"sync"

	"github.com/mysteriumnetwork/node/firewall/iptables"
	"github.com/mysteriumnetwork/node/utils"
	"github.com/mysteriumnetwork/node/utils/cmdutil"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type serviceIPTables struct {
	mu        sync.Mutex
	rules     []iptables.Rule
	ipForward serviceIPForward
}

const (
	chainForward     = "FORWARD"
	chainPreRouting  = "PREROUTING"
	chainPostRouting = "POSTROUTING"
)

// Setup sets NAT/Firewall rules for the given NATOptions.
func (svc *serviceIPTables) Setup(opts Options) (appliedRules []interface{}, err error) {
	log.Info().Msg("Setting up NAT/Firewall rules")
	svc.mu.Lock()
	defer svc.mu.Unlock()

	// Store applied rules so we can remove if setup exits prematurely (one of the latter rules fails to apply)
	var applied []iptables.Rule
	defer func() {
		if err == nil {
			return
		}
		log.Warn().Msg("Error detected, clearing up rules that were already setup")
		for _, rule := range applied {
			if err := svc.removeRule(rule); err != nil {
				log.Error().Err(err).Msg("Could not remove rule")
			}
		}
	}()

	for _, rule := range makeIPTablesRules(opts) {
		if err := svc.applyRule(rule); err != nil {
			return nil, err
		}
		applied = append(applied, rule)
	}
	log.Info().Msg("Setting up NAT/Firewall rules... done")
	return untypedIptRules(applied), nil
}

// Del removes given NAT/Firewall rules that were previously set up.
func (svc *serviceIPTables) Del(rules []interface{}) (err error) {
	log.Info().Msg("Deleting NAT/Firewall rules")
	svc.mu.Lock()
	defer svc.mu.Unlock()

	errs := utils.ErrorCollection{}
	for _, rule := range typedIptRules(rules) {
		log.Trace().Msgf("Deleting rule: %v", rule)
		if err := svc.removeRule(rule); err != nil {
			errs.Add(err)
		}
	}
	err = errs.Error()
	log.Info().Err(err).Msg("Deleting NAT/Firewall rules... done")
	return err
}

// Enable enables NAT service.
func (svc *serviceIPTables) Enable() error {
	err := svc.ipForward.Enable()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to enable IP forwarding")
	}
	return err
}

// Disable disables NAT service and deletes all rules.
func (svc *serviceIPTables) Disable() error {
	svc.ipForward.Disable()
	return svc.Del(untypedIptRules(svc.rules))
}

func (svc *serviceIPTables) applyRule(rule iptables.Rule) error {
	if err := iptablesExec(rule.ApplyArgs()...); err != nil {
		return err
	}
	svc.rules = append(svc.rules, rule)
	return nil
}

func (svc *serviceIPTables) removeRule(rule iptables.Rule) error {
	if err := iptablesExec(rule.RemoveArgs()...); err != nil {
		return err
	}
	for i := range svc.rules {
		if svc.rules[i].Equals(rule) {
			svc.rules = append(svc.rules[:i], svc.rules[i+1:]...)
			break
		}
	}
	return nil
}

func makeIPTablesRules(opts Options) (rules []iptables.Rule) {
	vpnNetwork := opts.VPNNetwork.String()

	if opts.EnableDNSRedirect {
		// DNS port redirect rule (udp)
		rule := iptables.AppendTo(chainPreRouting).RuleSpec(
			"--source", vpnNetwork, "--destination", opts.DNSIP.String(), "--protocol", "udp", "--dport", strconv.Itoa(53),
			"--jump", "REDIRECT",
			"--to-ports", strconv.Itoa(opts.DNSPort),
			"--table", "nat",
		)
		rules = append(rules, rule)

		// DNS port redirect rule (tcp)
		rule = iptables.AppendTo(chainPreRouting).RuleSpec(
			"--source", vpnNetwork, "--destination", opts.DNSIP.String(), "--protocol", "tcp", "--dport", strconv.Itoa(53),
			"--jump", "REDIRECT",
			"--to-ports", strconv.Itoa(opts.DNSPort),
			"--table", "nat",
		)
		rules = append(rules, rule)
	}

	// Protect private networks rule
	for _, ipNet := range protectedNetworks() {
		rule := iptables.AppendTo(chainForward).RuleSpec(
			"--source", vpnNetwork, "--destination", ipNet.String(),
			"--jump", "DROP")
		rules = append(rules, rule)
	}

	// NAT forwarding rule
	rule := iptables.AppendTo(chainPostRouting).RuleSpec("--source", vpnNetwork, "!", "--destination", vpnNetwork,
		"--jump", "SNAT", "--to", opts.ProviderExtIP.String(),
		"--table", "nat")
	rules = append(rules, rule)

	return rules
}

func iptablesExec(args ...string) error {
	args = append([]string{"/sbin/iptables"}, args...)
	if err := cmdutil.SudoExec(args...); err != nil {
		return errors.Wrap(err, "error calling IPTables")
	}
	return nil
}

func untypedIptRules(rules []iptables.Rule) []interface{} {
	res := make([]interface{}, len(rules))
	for i := range rules {
		res[i] = rules[i]
	}
	return res
}

func typedIptRules(rules []interface{}) []iptables.Rule {
	res := make([]iptables.Rule, len(rules))
	for i := range rules {
		res[i] = rules[i].(iptables.Rule)
	}
	return res
}
