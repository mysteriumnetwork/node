// +build android

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

package router

import (
	"net"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/utils/cmdutil"
)

type routingTable struct{}

func (t *routingTable) discoverGateway() (net.IP, error) {
	return nil, nil
}

func (t *routingTable) excludeRule(ip, gw net.IP) error {
	return nil
}

func (t *routingTable) deleteRule(ip, gw net.IP) error {
	return nil
}
