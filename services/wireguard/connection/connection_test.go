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
	"context"
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/core/ip"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
)

func TestConnectionStartStop(t *testing.T) {
	conn := newConn(t)

	// Start connection.
	sessionConfig, _ := json.Marshal(newServiceConfig())
	err := conn.Start(context.Background(), connection.ConnectOptions{
		Params:        connection.ConnectParams{DNS: "1.2.3.4"},
		SessionConfig: sessionConfig,
	})

	assert.NoError(t, err)
	assert.Equal(t, connectionstate.Connecting, <-conn.State())
	assert.Equal(t, connectionstate.Connected, <-conn.State())
	stats, err := conn.Statistics()
	assert.NoError(t, err)
	assert.EqualValues(t, 10, stats.BytesSent)
	assert.EqualValues(t, 11, stats.BytesReceived)

	// Stop connection.
	stopCh := make(chan struct{})
	go func() {
		// test auto-create connection during Reconnect
		conn.connectionEndpoint = nil
		err = conn.Reconnect(context.Background(), connection.ConnectOptions{
			Params:        connection.ConnectParams{DNS: "1.2.3.4"},
			SessionConfig: sessionConfig,
		})
		assert.NoError(t, err)
		conn.Stop()
		stopCh <- struct{}{}
	}()
	<-stopCh
	assert.NoError(t, err)
}

func TestConnectionStopAfterHandshakeError(t *testing.T) {
	conn := newConn(t)
	handshakeTimeoutErr := errors.New("handshake timeout")
	conn.handshakeWaiter = &mockHandshakeWaiter{err: handshakeTimeoutErr}
	sessionConfig, _ := json.Marshal(newServiceConfig())

	err := conn.Start(context.Background(), connection.ConnectOptions{SessionConfig: sessionConfig})
	assert.Error(t, handshakeTimeoutErr, err)
	assert.Equal(t, connectionstate.Connecting, <-conn.State())
	assert.Equal(t, connectionstate.Disconnecting, <-conn.State())
	assert.Equal(t, connectionstate.NotConnected, <-conn.State())
}

func TestConnectionStopOnceAfterHandshakeErrorAndStopCall(t *testing.T) {
	conn := newConn(t)
	handshakeTimeoutErr := errors.New("handshake timeout")
	conn.handshakeWaiter = &mockHandshakeWaiter{err: handshakeTimeoutErr}
	sessionConfig, _ := json.Marshal(newServiceConfig())

	err := conn.Start(context.Background(), connection.ConnectOptions{SessionConfig: sessionConfig})

	stopCh := make(chan struct{})
	go func() {
		conn.Stop()
		stopCh <- struct{}{}
	}()
	<-stopCh

	assert.Error(t, handshakeTimeoutErr, err)
	assert.Equal(t, connectionstate.Connecting, <-conn.State())
	assert.Equal(t, connectionstate.Disconnecting, <-conn.State())
	assert.Equal(t, connectionstate.NotConnected, <-conn.State())
}

func newConn(t *testing.T) *Connection {
	endpointFactory := func() (wg.ConnectionEndpoint, error) {
		return &mockConnectionEndpoint{}, nil
	}
	opts := Options{
		DNSScriptDir: "/dns/dir",
	}
	conn, err := NewConnection(opts, ip.NewResolverMock("172.44.1.12"), endpointFactory, &mockHandshakeWaiter{})
	assert.NoError(t, err)
	return conn.(*Connection)
}

func newServiceConfig() wg.ServiceConfig {
	endpoint, _ := net.ResolveUDPAddr("udp4", "127.0.0.1:51001")
	return wg.ServiceConfig{
		LocalPort:  51000,
		RemotePort: 51001,
		Provider: struct {
			PublicKey string
			Endpoint  net.UDPAddr
		}{
			PublicKey: "wg1",
			Endpoint:  *endpoint,
		},
		Consumer: struct {
			IPAddress net.IPNet
			DNSIPs    string
		}{
			IPAddress: net.IPNet{
				IP:   net.IPv4(127, 0, 0, 1),
				Mask: net.IPv4Mask(255, 255, 255, 128),
			},
			DNSIPs: "128.0.0.1",
		},
	}
}

type mockConnectionEndpoint struct{}

func (mce *mockConnectionEndpoint) ReconfigureConsumerMode(config wgcfg.DeviceConfig) error {
	return nil
}
func (mce *mockConnectionEndpoint) StartConsumerMode(config wgcfg.DeviceConfig) error { return nil }
func (mce *mockConnectionEndpoint) StartProviderMode(ip string, config wgcfg.DeviceConfig) error {
	return nil
}
func (mce *mockConnectionEndpoint) InterfaceName() string                { return "mce0" }
func (mce *mockConnectionEndpoint) Stop() error                          { return nil }
func (mce *mockConnectionEndpoint) Config() (wg.ServiceConfig, error)    { return wg.ServiceConfig{}, nil }
func (mce *mockConnectionEndpoint) AddPeer(_ string, _ wgcfg.Peer) error { return nil }
func (mce *mockConnectionEndpoint) RemovePeer(_ string) error            { return nil }
func (mce *mockConnectionEndpoint) ConfigureRoutes(_ net.IP) error       { return nil }
func (mce *mockConnectionEndpoint) PeerStats() (wgcfg.Stats, error) {
	return wgcfg.Stats{LastHandshake: time.Now(), BytesSent: 10, BytesReceived: 11}, nil
}
func (mce *mockConnectionEndpoint) Diag() bool {
	return true
}

type mockHandshakeWaiter struct {
	err error
}

func (m *mockHandshakeWaiter) Wait(ctx context.Context, statsFetch func() (wgcfg.Stats, error), timeout time.Duration, stop <-chan struct{}) error {
	return m.err
}
