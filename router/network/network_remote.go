/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package network

import (
	"fmt"
	"net"

	"github.com/mysteriumnetwork/node/supervisor/client"
)

// RoutingTableRemote implements a set of commands for supervisor deamon for creating,
// deleting and observe routing tables rules for a different needs.
type RoutingTableRemote struct{}

// DiscoverGateway returns system default gateway.
func (t *RoutingTableRemote) DiscoverGateway() (net.IP, error) {
	gw, err := client.Command("discover-gateway")
	if err != nil {
		return nil, fmt.Errorf("failed to discover gateway via supervisor: %w", err)
	}

	return net.ParseIP(gw), nil
}

// ExcludeRule adds a single IP address to be excluded from the main tunnelled traffic.
// Traffic sent to the IP address will be directed to the system default gaitway
// instead of tunnel.
func (t *RoutingTableRemote) ExcludeRule(ip, gw net.IP) error {
	_, err := client.Command("exclude-route", "-ip", ip.String(), "-gw", gw.String())
	if err != nil {
		return fmt.Errorf("failed to exclude route via supervisor: %w", err)
	}

	return nil
}

// DeleteRule removes excluded routing table rule to return it back to routing
// thought the tunnel.
func (t *RoutingTableRemote) DeleteRule(ip, gw net.IP) error {
	_, err := client.Command("delete-route", "-ip", ip.String(), "-gw", gw.String())
	if err != nil {
		return fmt.Errorf("failed to delete route via supervisor: %w", err)
	}

	return nil
}
