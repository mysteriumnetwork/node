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

package behavior

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/nat"
	"github.com/pion/stun"
)

// Enum of DiscoverNATMapping return values
const (
	MappingNone                 = "none"
	MappingIndependent          = "independent"
	MappingAddressDependent     = "address"
	MappingAddressPortDependent = "addressport"
)

// Enum of DiscoverNATFiltering return values
const (
	FilteringIndependent = "independent"
	FilteringAddress     = "address"
	FilteringAddressPort = "addressport"
)

// DefaultTimeout for each single STUN request.
const DefaultTimeout = 3 * time.Second

// STUN protocol compatibility errors
var (
	ErrResponseMessage = errors.New("error reading from response message channel")
	ErrNoXorAddress    = errors.New("no XOR-MAPPED-ADDRESS in message")
	ErrNoOtherAddress  = errors.New("no OTHER-ADDRESS in message")
)

// CHANGE-REQUEST value constants from RFC 5780 Section 7.2
var (
	changeRequestAddressPort = []byte{0x00, 0x00, 0x00, 0x06}
	changeRequestPort        = []byte{0x00, 0x00, 0x00, 0x02}
)

type stunServerConn struct {
	conn        *net.UDPConn
	LocalAddr   net.Addr
	RemoteAddr  *net.UDPAddr
	OtherAddr   *net.UDPAddr
	messageChan chan *stun.Message
	stopChan    chan struct{}
	stopOnce    sync.Once
}

func (c *stunServerConn) Close() error {
	c.stopOnce.Do(func() {
		close(c.stopChan)
	})
	return c.conn.Close()
}

// DiscoverNATBehavior returns either one of NATType* constants describing
// NAT behavior in practical sense for P2P connections or error
func DiscoverNATBehavior(ctx context.Context, address string, timeout time.Duration) (nat.NATType, error) {
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	mapping, err := DiscoverNATMapping(ctx, address, timeout)
	if err != nil {
		return "", err
	}
	switch mapping {
	case MappingAddressDependent, MappingAddressPortDependent:
		return nat.NATTypeSymmetric, nil
	case MappingNone:
		return nat.NATTypeNone, nil
	}

	filtering, err := DiscoverNATFiltering(ctx, address, timeout)
	if err != nil {
		return "", err
	}
	switch filtering {
	case FilteringIndependent:
		return nat.NATTypeFullCone, nil
	case FilteringAddress:
		return nat.NATTypeRestrictedCone, nil
	default:
		return nat.NATTypePortRestrictedCone, nil
	}
}

// DiscoverNATMapping returns either one of Mapping* constants describing
// NAT mapping behavior or error
func DiscoverNATMapping(ctx context.Context, address string, timeout time.Duration) (string, error) {
	mapTestConn, err := connect(address)
	if err != nil {
		return "", fmt.Errorf("STUN connection init failed: %w", err)
	}
	defer mapTestConn.Close()

	// Test I: Regular binding request
	request := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	ctx1, cl := context.WithTimeout(ctx, timeout)
	defer cl()
	resp, err := mapTestConn.roundTrip(ctx1, request, mapTestConn.RemoteAddr)
	if err != nil {
		return "", fmt.Errorf("mapping test I RT failed: %w", err)
	}

	// Parse response message for XOR-MAPPED-ADDRESS and make sure OTHER-ADDRESS valid
	resps1 := parse(resp)
	if resps1.xorAddr == nil {
		return "", ErrNoXorAddress
	}
	if resps1.otherAddr == nil {
		return "", ErrNoOtherAddress
	}
	addr, err := net.ResolveUDPAddr("udp4", resps1.otherAddr.String())
	if err != nil {
		return "", fmt.Errorf("other-address resolve failed: %w", err)
	}
	mapTestConn.OtherAddr = addr

	// Assert mapping behavior
	// TODO: it doesn't actually work because we bind at wildcard address
	// so condition is always false
	myIP, _, err := net.SplitHostPort(resps1.xorAddr.String())
	if err != nil {
		return "", fmt.Errorf("can't parse mirrored address: %w", err)
	}
	outboundIP, err := getOutboundIP(mapTestConn.RemoteAddr.String())
	if err != nil {
		return "", fmt.Errorf("can't get outbound address: %w", err)
	}
	if myIP == outboundIP.String() {
		return MappingNone, nil
	}

	// Test II: Send binding request to the other address but primary port
	ctx1, cl = context.WithTimeout(ctx, timeout)
	defer cl()
	oaddr := *mapTestConn.OtherAddr
	oaddr.Port = mapTestConn.RemoteAddr.Port
	resp, err = mapTestConn.roundTrip(ctx1, request, &oaddr)
	if err != nil {
		return "", fmt.Errorf("mapping test II RT failed: %w", err)
	}

	// Assert mapping behavior
	resps2 := parse(resp)
	if resps2.xorAddr.String() == resps1.xorAddr.String() {
		return MappingIndependent, nil
	}

	// Test III: Send binding request to the other address and port
	ctx1, cl = context.WithTimeout(ctx, timeout)
	defer cl()
	resp, err = mapTestConn.roundTrip(ctx1, request, mapTestConn.OtherAddr)
	if err != nil {
		return "", fmt.Errorf("mapping test III RT failed: %w", err)
	}

	// Assert mapping behavior
	resps3 := parse(resp)
	if resps3.xorAddr.String() == resps2.xorAddr.String() {
		return MappingAddressDependent, nil
	}

	return MappingAddressPortDependent, nil
}

// DiscoverNATFiltering returns either one of FILTERING_* constants describing
// NAT filtering behavior or error
func DiscoverNATFiltering(ctx context.Context, address string, timeout time.Duration) (string, error) {
	mapTestConn, err := connect(address)
	if err != nil {
		return "", fmt.Errorf("STUN connection init failed: %w", err)
	}
	defer mapTestConn.Close()

	// Test I: Regular binding request
	request := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	ctx1, cl := context.WithTimeout(ctx, timeout)
	defer cl()
	resp, err := mapTestConn.roundTrip(ctx1, request, mapTestConn.RemoteAddr)
	if err != nil {
		return "", fmt.Errorf("filtering test I RT failed: %w", err)
	}
	resps := parse(resp)
	if resps.xorAddr == nil {
		return "", fmt.Errorf("filtering test I got bad response: %w", ErrNoXorAddress)
	}

	if resps.otherAddr == nil {
		return "", fmt.Errorf("filtering test I got bad response: %w", ErrNoOtherAddress)
	}
	addr, err := net.ResolveUDPAddr("udp4", resps.otherAddr.String())
	if err != nil {
		return "", fmt.Errorf("other-address resolve failed: %w", err)
	}
	mapTestConn.OtherAddr = addr

	// Test II: Request to change both IP and port
	request = stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	request.Add(stun.AttrChangeRequest, changeRequestAddressPort)

	ctx1, cl = context.WithTimeout(ctx, timeout)
	defer cl()
	_, err = mapTestConn.roundTrip(ctx1, request, mapTestConn.RemoteAddr)
	switch {
	case err == nil:
		return FilteringIndependent, nil
	case err == ctx1.Err() && ctx.Err() == nil:
		// Nothing, just no response. Proceed to next test.
	default:
		return "", fmt.Errorf("filtering test II failed: %w", err)
	}

	// Test III: Request to change port only
	request = stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	request.Add(stun.AttrChangeRequest, changeRequestPort)

	ctx1, cl = context.WithTimeout(ctx, timeout)
	defer cl()
	_, err = mapTestConn.roundTrip(ctx1, request, mapTestConn.RemoteAddr)
	switch {
	case err == nil:
		return FilteringAddress, nil
	case err == ctx1.Err() && ctx.Err() == nil:
		return FilteringAddressPort, nil
	default:
		return "", fmt.Errorf("filtering test III failed: %w", err)
	}
}

// Parse a STUN message
func parse(msg *stun.Message) (ret struct {
	xorAddr    *stun.XORMappedAddress
	otherAddr  *stun.OtherAddress
	mappedAddr *stun.MappedAddress
	software   *stun.Software
}) {
	ret.mappedAddr = &stun.MappedAddress{}
	ret.xorAddr = &stun.XORMappedAddress{}
	ret.otherAddr = &stun.OtherAddress{}
	ret.software = &stun.Software{}
	if ret.xorAddr.GetFrom(msg) != nil {
		ret.xorAddr = nil
	}
	if ret.otherAddr.GetFrom(msg) != nil {
		ret.otherAddr = nil
	}
	if ret.mappedAddr.GetFrom(msg) != nil {
		ret.mappedAddr = nil
	}
	if ret.software.GetFrom(msg) != nil {
		ret.software = nil
	}
	return ret
}

// Given an address string, returns a stunServerConn
func connect(address string) (*stunServerConn, error) {
	addr, err := net.ResolveUDPAddr("udp4", address)
	if err != nil {
		return nil, err
	}

	c, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, err
	}
	serverConn := &stunServerConn{
		conn:        c,
		LocalAddr:   c.LocalAddr(),
		RemoteAddr:  addr,
		messageChan: make(chan *stun.Message),
		stopChan:    make(chan struct{}),
	}
	serverConn.listen()
	return serverConn, nil
}

// Send request and wait for response or timeout
func (c *stunServerConn) roundTrip(ctx context.Context, msg *stun.Message, addr net.Addr) (*stun.Message, error) {
	err := msg.NewTransactionID()
	if err != nil {
		return nil, err
	}
	_, err = c.conn.WriteTo(msg.Raw, addr)
	if err != nil {
		return nil, err
	}

	// Wait for response or timeout
	for {
		select {
		case m, ok := <-c.messageChan:
			if !ok {
				return nil, ErrResponseMessage
			}
			if m.TransactionID == msg.TransactionID {
				return m, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (c *stunServerConn) listen() {
	go func() {
		defer close(c.messageChan)
		for {
			buf := make([]byte, 1024)

			n, _, err := c.conn.ReadFromUDP(buf)
			if err != nil {
				if n == 0 {
					return
				}
				continue
			}
			buf = buf[:n]

			m := new(stun.Message)
			m.Raw = buf
			err = m.Decode()
			if err != nil {
				continue
			}

			select {
			case c.messageChan <- m:
			case <-c.stopChan:
				return
			}
		}
	}()
	return
}

func getOutboundIP(remoteAddr string) (net.IP, error) {
	dialer := net.Dialer{}

	conn, err := dialer.Dial("udp4", remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to determine outbound IP: %w", err)
	}
	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).IP, nil
}
