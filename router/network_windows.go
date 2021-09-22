// +build windows

/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package router

import (
	"fmt"
	"net"
	"os/exec"

	"github.com/jackpal/gateway"
)

type routingTable struct{}

func (t *routingTable) discoverGateway() (net.IP, error) {
	return gateway.DiscoverGateway()
}

func (t *routingTable) excludeRule(ip, gw net.IP) error {
	out, err := exec.Command("powershell", "-Command", "route add "+ip.String()+"/32 "+gw.String()).CombinedOutput()
	return fmt.Errorf("%s: %w", string(out), err)
}

func (t *routingTable) deleteRule(ip, gw net.IP) error {
	out, err := exec.Command("powershell", "-Command", "route delete "+ip.String()+"/32").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete route: %w, %s", err, string(out))
	}

	return nil
}
