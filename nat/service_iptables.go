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
	"sync"

	"github.com/mysteriumnetwork/node/utils"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type serviceIPTables struct {
	mu        sync.Mutex
	rules     map[RuleForwarding]struct{}
	ipForward serviceIPForward
}

func (service *serviceIPTables) Add(rule RuleForwarding) error {
	service.mu.Lock()
	defer service.mu.Unlock()

	if _, ok := service.rules[rule]; ok {
		return errors.New("rule already exists")
	}
	service.rules[rule] = struct{}{}

	err := iptables("append", rule)
	return errors.Wrap(err, "failed to add NAT forwarding rule")
}

func (service *serviceIPTables) Del(rule RuleForwarding) error {
	if err := iptables("delete", rule); err != nil {
		return err
	}

	service.mu.Lock()
	defer service.mu.Unlock()

	delete(service.rules, rule)
	return nil
}

func (service *serviceIPTables) Enable() error {
	err := service.ipForward.Enable()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to enable IP forwarding")
	}
	return err
}

func (service *serviceIPTables) Disable() (err error) {
	service.ipForward.Disable()

	service.mu.Lock()
	defer service.mu.Unlock()

	for rule := range service.rules {
		if delErr := iptables("delete", rule); delErr != nil && err == nil {
			err = delErr
		}
	}
	return err
}

func iptables(action string, rule RuleForwarding) error {
	arguments := "/sbin/iptables --table nat --" + action + " POSTROUTING --source " +
		rule.SourceSubnet + " ! --destination " +
		rule.SourceSubnet + " --jump SNAT --to " +
		rule.TargetIP
	cmd := utils.SplitCommand("sudo", arguments)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Warn().Err(err).Msgf("Failed to %s IP forwarding rule: %v Cmd output: %s", action, cmd.Args, string(output))
		return errors.Wrap(err, string(output))
	}

	log.Info().Msgf("Action %q applied for forwarding packets from %s to IP: %s", action, rule.SourceSubnet, rule.TargetIP)
	return nil
}
