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

package netutil

import (
	"net"

	"github.com/jackpal/gateway"
	"github.com/mysteriumnetwork/node/utils/cmdutil"
)

func AssignIP(iface string, subnet net.IPNet) error {
	if err := cmdutil.SudoExec("ip", "address", "replace", "dev", iface, subnet.String()); err != nil {
		return err
	}
	return cmdutil.SudoExec("ip", "link", "set", "dev", iface, "up")
}

func ExcludeRoute(ip net.IP) error {
	gw, err := gateway.DiscoverGateway()
	if err != nil {
		return err
	}

	return cmdutil.SudoExec("route", "add", "-host", ip.String(), gw.String())
}

func AddDefaultRoute(iface string) error {
	if err := cmdutil.SudoExec("route", "add", "-net", "0.0.0.0/1", "-interface", iface); err != nil {
		return err
	}

	return cmdutil.SudoExec("route", "add", "-net", "128.0.0.0/1", "-interface", iface)
}
