/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package tequilapi

import "net"

// NewListener returns tequilapi listener.
func NewListener(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}

// NewNoopListener returns noop tequilapi listener.
func NewNoopListener() (net.Listener, error) {
	return &noopListener{}, nil
}

type noopListener struct {
}

func (n noopListener) Accept() (net.Conn, error) {
	return nil, nil
}

func (n noopListener) Close() error {
	return nil
}

func (n noopListener) Addr() net.Addr {
	return nil
}
