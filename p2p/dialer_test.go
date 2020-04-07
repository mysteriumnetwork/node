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

package p2p

import (
	"context"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDialer_Exchange_And_Communication_When_Provider_With_PublicIP(t *testing.T) {
	consumerID, providerID, ks, cleanup := createTestIdentities(t)
	defer cleanup()

	signerFactory := func(id identity.Identity) identity.Signer {
		return identity.NewSigner(ks, identity.FromAddress(id.Address))
	}
	verifier := identity.NewVerifierSigned()
	brokerConn := nats.StartConnectionMock()
	defer brokerConn.Close()
	mockBroker := &mockBroker{conn: brokerConn}
	portPool := port.NewPool()
	providerPinger := &mockProviderNATPinger{}
	consumerPinger := &mockConsumerNATPinger{}
	ipResolver := ip.NewResolverMock("127.0.0.1")

	t.Run("Test provider listens to peer", func(t *testing.T) {
		channelListener := NewListener(mockBroker, "broker", signerFactory, verifier, ipResolver, providerPinger, portPool)
		err := channelListener.Listen(providerID, "wireguard", func(ch Channel) {
			ch.Handle("test", func(c Context) error {
				return c.OkWithReply(&Message{Data: []byte("pong")})
			})
		})
		require.NoError(t, err)
	})

	t.Run("Test consumer dialer creates new ready to use channel", func(t *testing.T) {
		channelDialer := NewDialer(mockBroker, "broker", signerFactory, verifier, ipResolver, consumerPinger, portPool)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		consumerChannel, err := channelDialer.Dial(ctx, consumerID, "wireguard", providerID)
		require.NoError(t, err)

		res, err := consumerChannel.Send(context.Background(), "test", &Message{Data: []byte("ping")})
		require.NoError(t, err)
		assert.Equal(t, "pong", string(res.Data))
	})
}

func TestDialer_Exchange_And_Communication_When_Provider_Behind_NAT(t *testing.T) {
	consumerID, providerID, ks, cleanup := createTestIdentities(t)
	defer cleanup()

	signerFactory := func(id identity.Identity) identity.Signer {
		return identity.NewSigner(ks, identity.FromAddress(id.Address))
	}
	verifier := identity.NewVerifierSigned()
	brokerConn := nats.StartConnectionMock()
	defer brokerConn.Close()
	mockBroker := &mockBroker{conn: brokerConn}
	portPool := &mockPortPool{}

	// Create mock NAT pinger.
	ports, err := acquirePorts(2)
	assert.NoError(t, err)
	providerPort := ports[0]
	consumerPort := ports[1]
	providerConn, err := net.DialUDP("udp", &net.UDPAddr{Port: providerPort}, &net.UDPAddr{Port: consumerPort})
	assert.NoError(t, err)
	consumerConn, err := net.DialUDP("udp", &net.UDPAddr{Port: consumerPort}, &net.UDPAddr{Port: providerPort})
	assert.NoError(t, err)
	providerPinger := &mockProviderNATPinger{conns: []*net.UDPConn{consumerConn, consumerConn}}
	consumerPinger := &mockConsumerNATPinger{conns: []*net.UDPConn{providerConn, providerConn}}
	// Simulate behind NAT behaviour with different IP's.
	ipResolver := ip.NewResolverMock("127.0.0.1", "1.1.1.1")

	t.Run("Test provider listens to peer", func(t *testing.T) {
		channelListener := NewListener(mockBroker, "broker", signerFactory, verifier, ipResolver, providerPinger, portPool)
		err = channelListener.Listen(providerID, "wireguard", func(ch Channel) {
			ch.Handle("test", func(c Context) error {
				return c.OkWithReply(&Message{Data: []byte("pong")})
			})
		})
		require.NoError(t, err)
	})

	t.Run("Test consumer dialer creates new ready to use channel", func(t *testing.T) {
		channelDialer := NewDialer(mockBroker, "broker", signerFactory, verifier, ipResolver, consumerPinger, portPool)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		consumerChannel, err := channelDialer.Dial(ctx, consumerID, "wireguard", providerID)
		require.NoError(t, err)

		res, err := consumerChannel.Send(context.Background(), "test", &Message{Data: []byte("ping")})
		require.NoError(t, err)
		assert.Equal(t, "pong", string(res.Data))
	})
}

func createTestIdentities(t *testing.T) (consumerID identity.Identity, providerID identity.Identity, ks *identity.Keystore, cleanup func()) {
	dir, err := ioutil.TempDir("", "p2pDialerTest")
	assert.NoError(t, err)
	cleanup = func() { os.RemoveAll(dir) }

	ks = identity.NewKeystoreFilesystem(dir, identity.NewMockKeystore(identity.MockKeys), identity.MockDecryptFunc)
	consumerAcc, err := ks.NewAccount("")
	assert.NoError(t, err)
	ks.Unlock(consumerAcc, "")
	consumerID = identity.FromAddress(consumerAcc.Address.Hex())
	providerAcc, err := ks.NewAccount("")
	assert.NoError(t, err)
	ks.Unlock(providerAcc, "")
	providerID = identity.FromAddress(providerAcc.Address.Hex())
	return
}

type mockConsumerNATPinger struct {
	conns []*net.UDPConn
}

func (m *mockConsumerNATPinger) PingProviderPeer(ctx context.Context, ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error) {
	return m.conns, nil
}

type mockProviderNATPinger struct {
	conns []*net.UDPConn
}

func (m *mockProviderNATPinger) PingConsumerPeer(ctx context.Context, ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error) {
	return m.conns, nil
}

type mockBroker struct {
	conn nats.Connection
}

func (m *mockBroker) Connect(serverURIs ...string) (nats.Connection, error) {
	return m.conn, nil
}

type mockPortPool struct {
}

func (m mockPortPool) Acquire() (port.Port, error) {
	return port.Port(0), nil
}

func (m mockPortPool) AcquireMultiple(n int) (ports []port.Port, err error) {
	for i := 0; i < n; i++ {
		ports = append(ports, port.Port(i))
	}
	return
}
