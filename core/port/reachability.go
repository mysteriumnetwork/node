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

package port

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"time"

	"github.com/gofrs/uuid"
)

const (
	PortFieldSize = 2
	UUIDSize      = uuid.Size
	PacketSize    = PortFieldSize + UUIDSize
	SendPackets   = 3
)

// GloballyReachable checks if UDP port is reachable from global Internet,
// performing probe against asymmetric UDP echo server
func GloballyReachable(ctx context.Context, port Port, echoServerAddress string, timeout time.Duration) (bool, error) {
	// Claim port
	rxAddr := &net.UDPAddr{
		Port: port.Num(),
	}

	rxSock, err := net.ListenUDP("udp", rxAddr)
	if err != nil {
		return false, err
	}
	defer rxSock.Close()

	// Send probe
	dialer := net.Dialer{}
	txSock, err := dialer.DialContext(ctx, "udp", echoServerAddress)
	if err != nil {
		return false, err
	}
	defer txSock.Close()

	msg := make([]byte, PacketSize)
	binary.BigEndian.PutUint16(msg, uint16(port.Num()))

	probeUUID, err := uuid.NewV4()
	if err != nil {
		return false, err
	}
	copy(msg[PortFieldSize:], probeUUID[:])

	for i := 0; i < 3; i++ {
		_, err = txSock.Write(msg)
		if err != nil && i == 0 {
			return false, err
		}
	}

	// Await response
	ctx1, cl := context.WithTimeout(ctx, timeout)
	defer cl()
	responseChan := make(chan struct{})

	// Background context-aware receiver
	go func() {
		buf := make([]byte, uuid.Size)
		for {
			n, _, err := rxSock.ReadFromUDP(buf)
			if err != nil {
				if n == 0 {
					return
				}
				continue
			}

			if n < uuid.Size {
				continue
			}

			if bytes.Equal(buf, probeUUID[:]) {
				select {
				case responseChan <- struct{}{}:
					return
				case <-ctx1.Done():
					return
				}
			}
		}
	}()

	// Either response will be receiver or not (context timeout)
	select {
	case <-responseChan:
		return true, nil
	case <-ctx1.Done():
		return false, nil
	}
}
