//go:build android

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

package mysterium

import (
	"fmt"
	"net"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"

	"github.com/mysteriumnetwork/node/p2p"
)

func peekLookAtSocketFd4(d *device.Device) (fd int, err error) {
	bind, ok := d.Bind().(*conn.StdNetBind)
	if !ok {
		return 0, fmt.Errorf("failed to peek socket fd")
	}

	return bind.PeekLookAtSocketFd4()
}

func peekLookAtSocketFd4From(c p2p.ServiceConn) (fd int, err error) {
	conn, ok := c.(*net.UDPConn)
	if !ok {
		return 0, fmt.Errorf("failed to peek socket fd")
	}

	sysconn, err := conn.SyscallConn()
	if err != nil {
		return
	}
	err = sysconn.Control(func(f uintptr) {
		fd = int(f)
	})
	if err != nil {
		return
	}
	return
}
