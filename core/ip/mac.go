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

package ip

import (
	"net"
	"strings"

	"github.com/pkg/errors"
)

// GetMACAddressForIP finds MAC address
func GetMACAddressForIP(ip string) (string, error) {
	var currentNetworkHardwareName string
	interfaces, _ := net.Interfaces()
	for _, i := range interfaces {

		if addresses, err := i.Addrs(); err == nil {
			for _, addr := range addresses {
				// only interested in the name with current IP address
				if strings.Contains(addr.String(), ip) {
					currentNetworkHardwareName = i.Name
				}
			}
		}
	}

	netInterface, err := net.InterfaceByName(currentNetworkHardwareName)

	if err != nil {
		return "", errors.Wrap(err, "failed to get MAC address")
	}

	macAddress, err := net.ParseMAC(netInterface.HardwareAddr.String())

	if err != nil {
		return "", errors.Wrap(err, "failed to parse MAC address")
	}

	return macAddress.String(), nil
}
