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
	"errors"
	"net"
)

func assignIP(iface string, subnet net.IPNet) error {
	return errors.New("not implemented")
}

func excludeRoute(ip net.IP) error {
	return errors.New("not implemented")
}

func addDefaultRoute(iface string) error {
	return errors.New("not implemented")
}
