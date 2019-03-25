/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package firewall

import (
	"errors"
	"fmt"

	"github.com/mysteriumnetwork/node/utils"
)

// AddInboundRule adds new inbound rule to the platform specific firewall.
func AddInboundRule(proto string, port int) error {
	name := fmt.Sprintf("myst-%d:%s", port, proto)
	cmd := fmt.Sprintf(`New-NetFirewallRule -DisplayName "%s" -Direction Inbound -LocalPort %d -Protocol %s -Action Allow`, name, port, proto)

	if isInboundRuleExist(name) {
		return nil
	}

	_, err := utils.PowerShell(cmd)
	return err
}

// RemoveInboundRule removes inbound rule from the platform specific firewall.
func RemoveInboundRule(proto string, port int) error {
	name := fmt.Sprintf("myst-%d:%s", port, proto)
	cmd := fmt.Sprintf(`Remove-NetFirewallRule -DisplayName "%s"`, name)

	if !isInboundRuleExist(name) {
		return errors.New("firewall rule not found")
	}

	_, err := utils.PowerShell(cmd)
	return err
}

func isInboundRuleExist(name string) bool {
	cmd := fmt.Sprintf(`Get-NetFirewallRule -DisplayName "%s"`, name)

	if _, err := utils.PowerShell(cmd); err != nil {
		return false
	}

	return true
}
