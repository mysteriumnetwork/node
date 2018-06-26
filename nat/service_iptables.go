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
	"github.com/mysterium/node/utils"
	"strings"
)

const natLogPrefix = "[nat] "

type serviceIPTables struct {
	rules   []RuleForwarding
	forward bool
}

func (service *serviceIPTables) Add(rule RuleForwarding) {
	service.rules = append(service.rules, rule)
}

func (service *serviceIPTables) Start() {
	service.clearStaleRules()
	service.enableIPForwarding()
	service.enableRules()
}

func (service *serviceIPTables) Stop() {
	service.disableRules()
	service.disableIPForwarding()
}

func (service *serviceIPTables) isIPForwardingEnabled() (enabled bool) {
	cmd := utils.SplitCommand("/sbin/sysctl", "-n net.ipv4.ip_forward")
	output, err := cmd.Output()
	if err != nil {
		log.Warn("Failed to check IP forwarding status: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
	}

	if strings.TrimSpace(string(output)) == "1" {
		log.Info(natLogPrefix, "IP forwarding already enabled")
		return true
	}
	return false
}

func (service *serviceIPTables) enableIPForwarding() {
	enabled := service.isIPForwardingEnabled()

	if enabled {
		service.forward = true
		return
	}
	cmd := utils.SplitCommand("sudo", "/sbin/sysctl -w net.ipv4.ip_forward=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Warn("Failed to enable IP forwarding: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
	} else {
		log.Info(natLogPrefix, "IP forwarding enabled")
	}
}

func (service *serviceIPTables) disableIPForwarding() {
	if service.forward {
		return
	}

	cmd := utils.SplitCommand("sudo", "/sbin/sysctl -w net.ipv4.ip_forward=0")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Warn("Failed to disable IP forwarding: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
	} else {
		log.Info(natLogPrefix, "IP forwarding disabled")
	}
}

func (service *serviceIPTables) enableRules() {
	for _, rule := range service.rules {
		arguments := "/sbin/iptables --table nat --append POSTROUTING --source " +
			rule.SourceAddress + " ! --destination " +
			rule.SourceAddress + " --jump SNAT --to " +
			rule.TargetIP
		cmd := utils.SplitCommand("sudo", arguments)
		if output, err := cmd.CombinedOutput(); err != nil {
			log.Warn("Failed to create ip forwarding rule: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
		} else {
			log.Info(natLogPrefix, "Forwarding packets from '", rule.SourceAddress, "' to IP: ", rule.TargetIP)
		}
	}
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
