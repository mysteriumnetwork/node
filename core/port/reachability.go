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
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/gofrs/uuid"
	"github.com/rs/zerolog/log"
)

const (
	portFieldSize = 2
	packetSize    = portFieldSize + uuid.Size
	sendPackets   = 3
)

// ErrEmptyServerAddressList indicates there are no servers to get response from
var ErrEmptyServerAddressList = errors.New("empty server address list specified")

// GloballyReachable checks if UDP port is reachable from global Internet,
// performing probe against asymmetric UDP echo server
func GloballyReachable(ctx context.Context, port Port, echoServerAddresses []string, timeout time.Duration) (bool, error) {
	count := len(echoServerAddresses)
	if count == 0 {
		return false, ErrEmptyServerAddressList
	}

	log.Debug().Msgf("Checking if port %d globally reachable via %v", port, echoServerAddresses)

	// Claim port
	rxAddr := &net.UDPAddr{
		Port: port.Num(),
	}

	rxSock, err := net.ListenUDP("udp", rxAddr)
	if err != nil {
		return false, err
	}
	defer rxSock.Close()

	// Prepare request
	msg := make([]byte, packetSize)
	binary.BigEndian.PutUint16(msg, uint16(port.Num()))

	probeUUID, err := uuid.NewV4()
	if err != nil {
		return false, err
	}
	copy(msg[portFieldSize:], probeUUID[:])

	// Send probes. Proceed to listen after first send success.
	sendResultChan := make(chan error)
	aggregatedSenderError := make(chan error)

	// Collect and aggregate output of senders. Yield result after first success.
	// Drain rest until all of them done.
	go func() {
		var (
			success bool
			lastErr error
		)
		for i := 0; i < count; i++ {
			lastErr = <-sendResultChan
			if lastErr == nil && !success {
				success = true
				aggregatedSenderError <- nil
			}
		}
		if !success {
			aggregatedSenderError <- lastErr
		}
	}()

	// Spawn senders
	for _, address := range echoServerAddresses {
		go func(echoServerAddress string) {
			sendResultChan <- sendProbe(ctx, echoServerAddress, msg)
		}(address)
	}

	if err := <-aggregatedSenderError; err != nil {
		return false, fmt.Errorf("every port probe send failed. last error: %w", err)
	}

	// Await response
	ctx, cl := context.WithTimeout(ctx, timeout)
	defer cl()
	responseChan := make(chan struct{})

	go probeReceiver(ctx, rxSock, probeUUID, responseChan)

	// Either response will be received or not. Both cases are valid results.
	select {
	case <-responseChan:
		return true, nil
	case <-ctx.Done():
		return false, nil
	}
}

func sendProbe(ctx context.Context, echoServerAddress string, msg []byte) error {
	dialer := net.Dialer{}
	txSock, err := dialer.DialContext(ctx, "udp", echoServerAddress)
	if err != nil {
		return err
	}
	defer txSock.Close()

	for i := 0; i < sendPackets; i++ {
		_, err = txSock.Write(msg)
		if err != nil && i == 0 {
			return err
		}
	}
	return nil
}

// receives datagrams from socket until one with correct probe UUID received.
// notifies caller via supplied channel.
func probeReceiver(ctx context.Context, rxSock *net.UDPConn, probeUUID uuid.UUID, responseChan chan<- struct{}) {
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
			case <-ctx.Done():
				return
			}
		}
	}
}
