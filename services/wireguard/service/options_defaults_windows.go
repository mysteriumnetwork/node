//+build windows

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

package service

import "net"

// DefaultOptions is a wireguard service configuration that will be used if no options provided.
var DefaultOptions = Options{
	ConnectDelay:   2000,
	PortMin:        0,
	MaxConnections: 253,
	Subnet: net.IPNet{
		IP:   net.ParseIP("10.182.0.0"),
		Mask: net.IPv4Mask(255, 255, 0, 0),
	}}
