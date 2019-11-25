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

	"github.com/mysteriumnetwork/node/config"
)

type servicePFCtl struct {
	mu        sync.Mutex
	rules     map[RuleForwarding]struct{}
	ipForward serviceIPForward
}

func (service *servicePFCtl) Add(rule RuleForwarding) error {
	service.mu.Lock()
	service.rules[rule] = struct{}{}
	service.mu.Unlock()

	return service.enableRules()
}

func (service *servicePFCtl) Del(rule RuleForwarding) error {
	service.mu.Lock()
	delete(service.rules, rule)
	service.mu.Unlock()

	return service.enableRules()
}

func (service *servicePFCtl) Enable() error {
	err := service.ipForward.Enable()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to enable IP forwarding")
	}

	return err
}

func (service *servicePFCtl) Disable() error {
	service.disableRules()
	service.ipForward.Disable()
	return nil
}

func ifaceByAddress(ipAddress string) (string, error) {
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
			if address.(*net.IPNet).IP.String() == ipAddress {
				return ifi.Name, nil
			}
		}
	}
	return "", errors.New("not able to determine outbound ethernet interface")
}

func (service *servicePFCtl) enableRules() error {
	service.mu.Lock()
	defer service.mu.Unlock()

	var natRule string

	for rule := range service.rules {
		iface, err := ifaceByAddress(rule.TargetIP)
		if err != nil {
			return err
		}

		destinations := config.GetString(config.FlagFirewallProtectedNetworks)
		if destinations == "" {
			log.Info().Msgf("no protected networks set")
		} else {
			natRule += fmt.Sprintf("no nat on %s inet from %s to { %s } \n",
				iface, rule.SourceSubnet, destinations)
		}
		natRule += fmt.Sprintf("nat on %v inet from %v to any -> %v\n", iface, rule.SourceSubnet, rule.TargetIP)
	}

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
