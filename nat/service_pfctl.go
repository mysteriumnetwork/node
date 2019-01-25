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
	"net"
	"os/exec"
	"strings"
	"sync"

	log "github.com/cihub/seelog"
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
		log.Warn(natLogPrefix, "Failed to enable IP forwarding: ", err)
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
	for rule := range service.rules {
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
