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
	"errors"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/utils"
	"net"
	"os/exec"
	"strings"
)

type servicePFCtl struct {
	rules   []RuleForwarding
	forward bool
}

func (service *servicePFCtl) Add(rule RuleForwarding) {
	service.rules = append(service.rules, rule)
}

func (service *servicePFCtl) Start() error {
	service.clearStaleRules()
	err := service.enableIPForwarding()
	if err != nil {
		return err
	}
	err = service.enableRules()
	if err != nil {
		return err
	}
	return nil
}

func (service *servicePFCtl) Stop() {
	service.disableRules()
	service.disableIPForwarding()
}

func (service *servicePFCtl) isIPForwardingEnabled() (enabled bool) {
	cmd := utils.SplitCommand("/usr/sbin/sysctl", "-n net.inet.ip.forwarding")
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

func (service *servicePFCtl) enableIPForwarding() error {
	enabled := service.isIPForwardingEnabled()

	if enabled {
		service.forward = true
		return nil
	}
	cmd := utils.SplitCommand("/usr/sbin/sysctl", "-w net.inet.ip.forwarding=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Warn("Failed to enable IP forwarding: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
		return err
	}

	log.Info(natLogPrefix, "IP forwarding enabled")
	return nil
}

func (service *servicePFCtl) disableIPForwarding() {
	if service.forward {
		return
	}

	cmd := utils.SplitCommand("/usr/sbin/sysctl", "-w net.inet.ip.forwarding=0")

	if output, err := cmd.CombinedOutput(); err != nil {
		log.Warn("Failed to disable IP forwarding: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
	} else {
		log.Info(natLogPrefix, "IP forwarding disabled")
	}
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
	for _, rule := range service.rules {
		iface, err := ifaceByAddress(rule.TargetIP)
		if err != nil {
			return err
		}
		natRule := fmt.Sprintf("nat on %v inet from %v to any -> %v", iface, rule.SourceAddress, rule.TargetIP)
		arguments := fmt.Sprintf(`echo "%v" | /sbin/pfctl -vEf -`, natRule)
		cmd := exec.Command(
			"sh",
			"-c",
			arguments,
		)

		if output, err := cmd.CombinedOutput(); err != nil {
			if !strings.Contains(string(output), natRule) {
				log.Warn("Failed to create pfctl rule: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
				return err
			}
		}

		log.Info(natLogPrefix, "NAT rule from '", rule.SourceAddress, "' to IP: ", rule.TargetIP, " added")
	}
	return nil
}

func (service *servicePFCtl) disableRules() {
	cmd := utils.SplitCommand("/sbin/pfctl", "-F nat")

	if output, err := cmd.CombinedOutput(); err != nil {
		log.Warn("Failed cleanup pfctl rules: ", cmd.Args, " Returned exit error: ", err.Error(), " Cmd output: ", string(output))
	}

	log.Info(natLogPrefix, "NAT rules cleared")
}

func (service *servicePFCtl) clearStaleRules() {
	service.disableRules()
}
