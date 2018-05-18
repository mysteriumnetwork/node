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
	"os/exec"

	log "github.com/cihub/seelog"
	"strings"
)

const NatLogPrefix = "[nat] "

type serviceIPTables struct {
	rules   []RuleForwarding
	forward bool
}

func (service *serviceIPTables) Add(rule RuleForwarding) {
	service.rules = append(service.rules, rule)
}

func (service *serviceIPTables) Start() error {
	service.clearStaleRules()

	if err := service.enableIPForwarding(); err != nil {
		return err
	}
	if err := service.enableRules(); err != nil {
		service.disableIPForwarding()
		return err
	}

	return nil
}

func (service *serviceIPTables) Stop() error {
	if err := service.disableRules(); err != nil {
		return err
	}
	if err := service.disableIPForwarding(); err != nil {
		return err
	}

	return nil
}

func (service *serviceIPTables) isIPForwardingEnabled() (enabled bool, err error) {
	out, err := exec.Command("sysctl", "-n", "net.ipv4.ip_forward").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("Failed to check IP forwarding status: %s", err)
	}

	if strings.TrimSpace(string(out)) == "1" {
		log.Info(NatLogPrefix, "IP forwarding already enabled")
		return true, nil
	}
	return false, nil
}

func (service *serviceIPTables) enableIPForwarding() (err error) {

	enabled, err := service.isIPForwardingEnabled()
	if err != nil {
		return err
	}

	if enabled {
		service.forward = true
		return nil
	}

	cmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to enable IP forwarding: %s", err)
	}

	log.Info(NatLogPrefix, "IP forwarding enabled")
	return nil
}

func (service *serviceIPTables) disableIPForwarding() (err error) {
	if service.forward {
		return nil
	}

	cmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=0")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to disable IP forwarding. %s", err)
	}

	log.Info(NatLogPrefix, "IP forwarding disabled")
	return nil
}

func (service *serviceIPTables) enableRules() error {
	for _, rule := range service.rules {
		cmd := exec.Command(
			"iptables",
			"--table", "nat",
			"--append", "POSTROUTING",
			"--source", rule.SourceAddress,
			"!", "--destination", rule.SourceAddress,
			"--jump", "SNAT",
			"--to", rule.TargetIP,
		)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("Failed to create ip forwarding rule: %s. %s", cmd.Args, err.Error())
		}
		log.Info(NatLogPrefix, "Forwarding packets from '", rule.SourceAddress, "' to IP: ", rule.TargetIP)
	}
	return nil
}

func (service *serviceIPTables) disableRules() error {
	for _, rule := range service.rules {
		cmd := exec.Command(
			"iptables",
			"--table", "nat",
			"--delete", "POSTROUTING",
			"--source", rule.SourceAddress,
			"!", "--destination", rule.SourceAddress,
			"--jump", "SNAT",
			"--to", rule.TargetIP,
		)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("Failed to delete ip forwarding rule: %s. %s", cmd.Args, err.Error())
		}
		log.Info(NatLogPrefix, "Stopped forwarding packets from '", rule.SourceAddress, "' to IP: ", rule.TargetIP)
	}
	return nil
}

func (service *serviceIPTables) clearStaleRules() {
	service.disableRules()
}
