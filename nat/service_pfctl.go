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
	"os/exec"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/utils"
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
		natRule += fmt.Sprintf("no nat on %s inet from %s to { %s } \n",
			iface, rule.SourceSubnet, config.GetString(config.FlagFirewallProtectedNetworks))
		natRule += fmt.Sprintf("nat on %v inet from %v to any -> %v\n", iface, rule.SourceSubnet, rule.TargetIP)
	}

	arguments := fmt.Sprintf(`echo "%v" | /sbin/pfctl -vEf -`, natRule)
	cmd := exec.Command(
		"sh",
		"-c",
		arguments,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(output), natRule) {
			log.Warn().Err(err).Msgf("Failed to create pfctl rule: %v returned exit error. Cmd output: %s", cmd.Args, string(output))
			return err
		}
	}
	log.Info().Msg("NAT rules applied")

	return nil
}

func (service *servicePFCtl) disableRules() {
	cmd := utils.SplitCommand("/sbin/pfctl", "-F nat")

	if output, err := cmd.CombinedOutput(); err != nil {
		log.Warn().Err(err).Msgf("Failed cleanup pfctl rules: %v returned exit error. Cmd output: %s", cmd.Args, string(output))
	}

	log.Info().Msg("NAT rules cleared")
}
