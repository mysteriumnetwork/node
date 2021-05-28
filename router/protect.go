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
	"sync"
)

var (
	mu      sync.RWMutex
	protect func(fd int) error = nil
)

// SetProtectFunc sets the callback for using to protect provided connection from going through the tunnel.
func SetProtectFunc(f func(fd int) error) {
	mu.Lock()
	defer mu.Unlock()

	protect = f
}

// Protect protects provided connection from going through the tunnel.
func Protect(fd int) error {
	mu.RLock()
	defer mu.RUnlock()

	if protect == nil {
		return nil
	}

	return protect(fd)
}

// ProtectUDPConn protects provided UDP connection from going through the tunnel.
func ProtectUDPConn(c *net.UDPConn) error {
	sysconn, err := c.SyscallConn()
	if err != nil {
		return err
	}

	fd := 0

	err = sysconn.Control(func(f uintptr) {
		fd = int(f)
	})
	if err != nil {
		return err
	}

	return Protect(fd)
}
