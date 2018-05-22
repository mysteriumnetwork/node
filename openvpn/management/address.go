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

package management

import (
	"net"
	"fmt"
	"strings"
	"strconv"
	"errors"
)

/**
 * Used for OpenVPN management connection
 */
type Address struct {
	IP   string
	Port int
}

func (ma *Address) String() string {
	return fmt.Sprintf("%s:%d", ma.IP, ma.Port)
}

/*
 * Finds an available TCP port on 127.0.0.1
 *
 * OpenVPN management connection needs to listen on a known port
 * using this method we can figure out which port/address to use
 */
func GetAvailableAddress() (*Address, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, errors.New("Failed to open a connection for port discovery.")
	}

	defer l.Close()

	// get the actual bound address/port
	address := l.Addr().(*net.TCPAddr)

	return &Address{address.IP.String(), address.Port}, nil
}

func GetPortAndAddressFromString(address string) (*Address, error) {
	str := strings.Split(address, ":")

	if len(str) < 2 {
		return nil, errors.New("Failed to parse address string.")
	}

	port, err := strconv.Atoi(str[1])

	if err != nil {
		return nil, errors.New("Failed to parse port number.")
	}

	return &Address{str[0], port}, nil
}
