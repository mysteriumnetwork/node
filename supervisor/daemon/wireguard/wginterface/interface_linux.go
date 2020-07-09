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

package wginterface

import (
	"errors"
	"net"

	"golang.zx2c4.com/wireguard/tun"
)

func createTunnel(requestedInterfaceName string) (tunnel tun.Device, interfaceName string, err error) {
	return nil, requestedInterfaceName, errors.New("not implemented")
}

func newUAPIListener(interfaceName string) (listener net.Listener, err error) {
	return nil, errors.New("not implemented")
}

func applySocketPermissions(interfaceName string, uid string) error {
	return errors.New("not implemented")
}
