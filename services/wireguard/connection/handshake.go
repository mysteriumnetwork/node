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

package connection

import (
	"errors"
	"net"
	"time"

	"github.com/mysteriumnetwork/node/services/wireguard"
)

// HandshakeWaiter waits for handshake.
type HandshakeWaiter interface {
	// Wait waits until WireGuard does initial handshake.
	Wait(statsFetch func() (*wireguard.Stats, error), timeout time.Duration, stop <-chan struct{}) error
}

// NewHandshakeWaiter returns handshake waiter instance.
func NewHandshakeWaiter() HandshakeWaiter {
	return &handshakeWaiter{}
}

type handshakeWaiter struct {
}

func (h *handshakeWaiter) Wait(statsFetch func() (*wireguard.Stats, error), timeout time.Duration, stop <-chan struct{}) error {
	// We need to send any packet to initialize handshake process.
	handshakePingConn, err := net.DialTimeout("tcp", "8.8.8.8:53", 100*time.Millisecond)
	if err == nil {
		defer handshakePingConn.Close()
	}
	timeoutCh := time.After(timeout)
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
		case <-timeoutCh:
			return errors.New("failed to receive initial handshake")
		case <-stop:
			return errors.New("stop received")
		}
	}
}
