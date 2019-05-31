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

package ip

import "net"

// Resolver allows resolving current IP
type Resolver interface {
	GetPublicIP() (string, error)
	GetOutboundIP() (string, error)
}

// delcared as var for override in test
var checkAddress = "8.8.8.8:53"

// GetOutbound provides an outbound IP address of the current system.
func GetOutbound() (net.IP, error) {
	conn, err := net.Dial("udp4", checkAddress)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).IP, nil
}
