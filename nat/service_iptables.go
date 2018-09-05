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
	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/utils"
)

const natLogPrefix = "[nat] "

type serviceIPTables struct {
	rules     []RuleForwarding
	ipForward serviceIPForward
}

func (service *serviceIPTables) Add(rule RuleForwarding) {
	service.rules = append(service.rules, rule)
}

func (service *serviceIPTables) Start() error {
	err := service.ipForward.Enable()
	if err != nil {
		return err
	}

	service.clearStaleRules()
	err = service.enableRules()
	if err != nil {
		return err
	}
	return nil
}

func (service *serviceIPTables) Stop() {
	service.disableRules()
	service.ipForward.Disable()
}

func (service *serviceIPTables) enableRules() error {
	for _, rule := range service.rules {
		arguments := "/sbin/iptables --table nat --append POSTROUTING --source " +
			rule.SourceAddress + " ! --destination " +
			rule.SourceAddress + " --jump SNAT --to " +
			rule.TargetIP
		cmd := utils.SplitCommand("sudo", arguments)
		if output, err := cmd.CombinedOutput(); err != nil {
			log.Warn("Failed to create ip forwarding rule: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
			return err
		}

		log.Info(natLogPrefix, "Forwarding packets from '", rule.SourceAddress, "' to IP: ", rule.TargetIP)
	}
	return nil
}

func (service *serviceIPTables) disableRules() {
	for _, rule := range service.rules {
		arguments := "/sbin/iptables --table nat --delete POSTROUTING --source " +
			rule.SourceAddress + " ! --destination " +
			rule.SourceAddress + " --jump SNAT --to " +
			rule.TargetIP
		cmd := utils.SplitCommand("sudo", arguments)
		if output, err := cmd.CombinedOutput(); err != nil {
			log.Warn("Failed to delete ip forwarding rule: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
		} else {
			log.Info(natLogPrefix, "Stopped forwarding packets from '", rule.SourceAddress, "' to IP: ", rule.TargetIP)
		}
	}
}

func (service *serviceIPTables) clearStaleRules() {
	service.disableRules()
}
