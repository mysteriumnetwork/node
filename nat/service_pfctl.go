/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/mysteriumnetwork/node/utils/cmdutil"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type servicePFCtl struct {
	mu        sync.Mutex
	rules     []string
	ipForward serviceIPForward
}

// Setup sets NAT/Firewall rules for the given NATOptions.
func (service *servicePFCtl) Setup(opts Options) (appliedRules []interface{}, err error) {
	log.Info().Msg("Setting up NAT/Firewall rules")
	service.mu.Lock()
	defer service.mu.Unlock()

	rules, err := makePfctlRules(opts)
	if err != nil {
		return nil, err
	}

	if err := service.pfctlExec(rules); err != nil {
		return nil, err
	}
	service.rules = append(service.rules, rules...)

	return untypedPfctlRules(rules), nil
}

func (service *servicePFCtl) Del(rules []interface{}) error {
	log.Info().Msg("Deleting NAT/Firewall rules")
	service.mu.Lock()
	defer service.mu.Unlock()

	rulesToDel := typedPfctlRules(rules)
	var newRules []string
	for _, rule := range service.rules {
		shouldDelete := false
		for _, ruleToDel := range rulesToDel {
			if rule == ruleToDel {
				shouldDelete = true
				break
			}
		}

		if !shouldDelete {
			newRules = append(newRules, rule)
		}
	}

	if err := service.pfctlExec(newRules); err != nil {
		return err
	}

	service.rules = newRules
	log.Info().Msg("Deleting NAT/Firewall rules... done")
	return nil
}

func makePfctlRules(opts Options) (rules []string, err error) {
	externalIface, err := ifaceByAddress(opts.ProviderExtIP)
	if err != nil {
		return nil, err
	}

	if opts.EnableDNSRedirect {
		// DNS port redirect rule
		tunnelIface, err := ifaceByAddress(opts.DNSIP)
		if err != nil {
			return nil, err
		}
		rule := fmt.Sprintf("rdr pass on %s inet proto { udp, tcp } from any to %s port 53 -> %s port %d",
			tunnelIface,
			opts.VPNNetwork.String(),
			opts.DNSIP,
			opts.DNSPort,
		)
		rules = append(rules, rule)
	}

	// Protect private networks rule
	networks := protectedNetworks()
	if len(networks) > 0 {
		var targets []string
		for _, network := range networks {
			targets = append(targets, network.String())
		}
		rule := fmt.Sprintf("no nat on %s inet from %s to { %s }",
			externalIface,
			opts.VPNNetwork.String(),
			strings.Join(targets, ", "),
		)
		rules = append(rules, rule)
	}

	// NAT forwarding rule
	rule := fmt.Sprintf("nat on %s inet from %s to any -> %s",
		externalIface,
		opts.VPNNetwork.String(),
		opts.ProviderExtIP,
	)
	rules = append(rules, rule)

	return rules, nil
}

// Enable enables NAT service.
func (service *servicePFCtl) Enable() error {
	err := service.ipForward.Enable()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to enable IP forwarding")
	}
	return err
}

// Disable disables NAT service and deletes all rules.
func (service *servicePFCtl) Disable() error {
	service.mu.Lock()
	defer service.mu.Unlock()

	service.ipForward.Disable()
	service.rules = nil
	service.disableRules()
	return nil
}

func ifaceByAddress(ip net.IP) (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, ifi := range ifaces {
		addresses, err := ifi.Addrs()
		if err != nil {
			return "", err
		}
		for _, address := range addresses {
			if address.(*net.IPNet).Contains(ip) {
				return ifi.Name, nil
			}
		}
	}
	return "", errors.New("not able to determine outbound ethernet interface for IP: " + ip.String())
}

func (service *servicePFCtl) pfctlExec(rules []string) error {
	natRule := strings.Join(rules, "\n")
	arguments := fmt.Sprintf(`echo "%v" | /sbin/pfctl -vEf -`, natRule)

	if output, err := cmdutil.ExecOutput("sh", "-c", arguments); err != nil {
		if !strings.Contains(output, natRule) {
			log.Warn().Err(err).Msgf("Failed to create pfctl rule")
			return err
		}
	}
	log.Info().Msg("NAT rules applied")
	return nil
}

func (service *servicePFCtl) disableRules() {
	_, err := cmdutil.ExecOutput("/sbin/pfctl", "-F", "nat")
	if err != nil {
		log.Warn().Err(err).Msgf("Failed cleanup NAT rules (pfctl)")
	} else {
		log.Info().Msg("NAT rules cleared")
	}
}

func untypedPfctlRules(rules []string) []interface{} {
	res := make([]interface{}, len(rules))
	for i := range rules {
		res[i] = rules[i]
	}
	return res
}

func typedPfctlRules(rules []interface{}) []string {
	res := make([]string, len(rules))
	for i := range rules {
		res[i] = rules[i].(string)
	}
	return res
}
