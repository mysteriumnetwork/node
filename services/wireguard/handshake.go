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

package wireguard

import (
	"errors"
	"net"
	"time"
)

const handshakeTimeout = 2 * time.Minute

// WaitHandshake waits until WireGuard does initial handshake.
func WaitHandshake(statsFetch func() (*Stats, error), stop chan struct{}) error {
	// We need to send any packet to initialize handshake process.
	handshakePingConn, err := net.DialTimeout("tcp", "8.8.8.8:53", 100*time.Millisecond)
	if err == nil {
		defer handshakePingConn.Close()
	}
	timeout := time.After(handshakeTimeout)
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			stats, err := statsFetch()
			if err != nil {
				return err
			}
			if !stats.LastHandshake.IsZero() {
				return nil
			}
		case <-timeout:
			return errors.New("failed to receive initial handshake")
		case <-stop:
			return errors.New("stop received")
		}
	}
}
