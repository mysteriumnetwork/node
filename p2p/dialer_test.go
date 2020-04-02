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

func TestDialerExchangeAndCommunication(t *testing.T) {
	dir, err := ioutil.TempDir("", "p2pDialerTest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewKeystoreFilesystem(dir, identity.NewMockKeystore(identity.MockKeys), identity.MockDecryptFunc)
	consumerAcc, err := ks.NewAccount("")
	assert.NoError(t, err)
	ks.Unlock(consumerAcc, "")
	consumerID := identity.FromAddress(consumerAcc.Address.Hex())
	providerAcc, err := ks.NewAccount("")
	assert.NoError(t, err)
	ks.Unlock(providerAcc, "")
	providerID := identity.FromAddress(providerAcc.Address.Hex())
	signerFactory := func(id identity.Identity) identity.Signer {
		return identity.NewSigner(ks, identity.FromAddress(id.Address))
	}
	verifier := identity.NewVerifierSigned()
	brokerConn := nats.StartConnectionMock()
	defer brokerConn.Close()
	mockBroker := &mockBroker{conn: brokerConn}

	mockPortPool := &mockPortPool{}

	ports := acquirePorts(t, 2)
	providerPort := ports[0]
	consumerPort := ports[1]
	providerConn, err := net.DialUDP("udp", &net.UDPAddr{Port: providerPort}, &net.UDPAddr{Port: consumerPort})
	assert.NoError(t, err)
	consumerConn, err := net.DialUDP("udp", &net.UDPAddr{Port: consumerPort}, &net.UDPAddr{Port: providerPort})
	assert.NoError(t, err)
	providerPinger := &mockProviderNATPinger{conns: []*net.UDPConn{consumerConn, consumerConn}}
	consumerPinger := &mockConsumerNATPinger{conns: []*net.UDPConn{providerConn, providerConn}}

	ipResolver := ip.NewResolverMock("127.0.0.1")

	providerChannelReady := make(chan struct{}, 1)
	t.Run("Test provider listens to peer", func(t *testing.T) {
		channelListener := NewListener(mockBroker, "broker", signerFactory, verifier, ipResolver, providerPinger, mockPortPool)
		err = channelListener.Listen(providerID, "wireguard", func(ch Channel) {
			ch.Handle("test", func(c Context) error {
				return c.OkWithReply(&Message{Data: []byte("pong")})
			})
			providerChannelReady <- struct{}{}
		})
		require.NoError(t, err)
	})

	t.Run("Test consumer dialer creates new ready to use channel", func(t *testing.T) {
		channelDialer := NewDialer(mockBroker, "broker", signerFactory, verifier, ipResolver, consumerPinger, mockPortPool)

		consumerChannel, err := channelDialer.Dial(consumerID, providerID, "wireguard", 5*time.Second)
		require.NoError(t, err)

		// TODO: This should be removed and fixed with https://github.com/mysteriumnetwork/node/issues/1987
		<-providerChannelReady

		res, err := consumerChannel.Send("test", &Message{Data: []byte("ping")})
		require.NoError(t, err)
		assert.Equal(t, "pong", string(res.Data))
	})
}

type mockConsumerNATPinger struct {
	conns []*net.UDPConn
}

func (m *mockConsumerNATPinger) PingProviderPeer(ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error) {
	return m.conns, nil
}

type mockProviderNATPinger struct {
	conns []*net.UDPConn
}

func (m *mockProviderNATPinger) PingConsumerPeer(ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error) {
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
