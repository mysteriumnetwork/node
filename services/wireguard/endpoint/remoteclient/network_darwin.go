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

package remoteclient

import (
	"net"

	supervisorclient "github.com/mysteriumnetwork/node/supervisor/client"
)

func assignIP(iface string, subnet net.IPNet) error {
	_, err := supervisorclient.Command("assign-ip", "-iface", iface, "-net", subnet.String())
	return err
}

func excludeRoute(ip net.IP) error {
	_, err := supervisorclient.Command("exclude-route", "-ip", ip.String())
	return err
}

func addDefaultRoute(iface string) error {
	_, err := supervisorclient.Command("default-route", "-iface", iface)
	return err
}
