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
	"fmt"
	"os/exec"

	"bytes"
	log "github.com/cihub/seelog"
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

	return service.disableIPForwarding()
}

func (service *serviceIPTables) isIPForwardingEnabled() (enabled bool, err error) {
	out, err := exec.Command("/sbin/sysctl", "-n", "net.ipv4.ip_forward").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("Failed to check IP forwarding status: %s", err)
	}

	if strings.TrimSpace(string(out)) == "1" {
		log.Info(natLogPrefix, "IP forwarding already enabled")
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

	cmd := exec.Command(
		"sh",
		"-c",
		"sudo /sbin/sysctl -w net.ipv4.ip_forward=1",
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to enable IP forwarding: %s", err)
	}

	log.Info(natLogPrefix, "IP forwarding enabled")
	return nil
}

func (service *serviceIPTables) disableIPForwarding() (err error) {
	if service.forward {
		return nil
	}

	cmd := exec.Command(
		"sh",
		"-c",
		"sudo /sbin/sysctl -w net.ipv4.ip_forward=0",
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to disable IP forwarding. %s", err)
	}

	log.Info(natLogPrefix, "IP forwarding disabled")
	return nil
}

func (service *serviceIPTables) enableRules() error {
	var stderr bytes.Buffer

	for _, rule := range service.rules {
		cmd := exec.Command(
			"sh",
			"-c",
			"sudo /sbin/iptables --table nat --append POSTROUTING --source "+
				rule.SourceAddress+" ! --destination "+
				rule.SourceAddress+" --jump SNAT --to "+
				rule.TargetIP,
		)
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("Failed to create ip forwarding rule: %s. %s, cause: %s", cmd.Args, err.Error(), stderr.String())
		}

		log.Info(natLogPrefix, "Forwarding packets from '", rule.SourceAddress, "' to IP: ", rule.TargetIP)

	}
	return nil
}

func (service *serviceIPTables) disableRules() error {

	for _, rule := range service.rules {
		cmd := exec.Command(
			"sudo",
			"/sbin/iptables",
			"--table", "nat",
			"--delete", "POSTROUTING", "--source",
			rule.SourceAddress, "!", "--destination",
			rule.SourceAddress, "--jump", "SNAT", "--to",
			rule.TargetIP,
		)
		if output, err := cmd.CombinedOutput(); err != nil {
			log.Warn("Failed to delete ip forwarding rule: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
		}
		log.Info(natLogPrefix, "Stopped forwarding packets from '", rule.SourceAddress, "' to IP: ", rule.TargetIP)
	}
	return nil
}

func (service *serviceIPTables) clearStaleRules() {
	service.disableRules()
}
