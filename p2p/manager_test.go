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

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
)

func TestManagerExchangeAndCommunication(t *testing.T) {
	dir, err := ioutil.TempDir("", "p2pManagerTest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	ks := keystore.NewKeyStore(dir, keystore.LightScryptN, keystore.LightScryptP)
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

	ports := acquirePorts(t, 2)
	providerPort := ports[0]
	consumerPort := ports[1]
	providerConn, err := net.DialUDP("udp", &net.UDPAddr{Port: providerPort}, &net.UDPAddr{Port: consumerPort})
	assert.NoError(t, err)
	consumerConn, err := net.DialUDP("udp", &net.UDPAddr{Port: consumerPort}, &net.UDPAddr{Port: providerPort})
	assert.NoError(t, err)
	providerPinger := &mockProviderNATPinger{conns: []*net.UDPConn{consumerConn, nil}}
	consumerPinger := &mockConsumerNATPinger{conns: []*net.UDPConn{providerConn, nil}}

	ipResolver := ip.NewResolverMock("127.0.0.1")

	t.Run("Test provider subscribes to channels requests", func(t *testing.T) {
		providerChannelSubscriber := Manager{
			portPool:       port.NewPool(),
			broker:         mockBroker,
			providerPinger: providerPinger,
			pendingConfigs: map[PublicKey]*p2pConnectConfig{},
			signer:         signerFactory,
			verifier:       verifier,
			ipResolver:     ipResolver,
			brokerAddress:  "mock",
		}
		err = providerChannelSubscriber.SubscribeChannel(providerID, func(ch *Channel) {
			ch.Handle("test", func(c Context) error {
				return c.OkWithReply(&Message{Data: []byte("pong")})
			})
		})
		assert.NoError(t, err)
	})

	t.Run("Test consumer exchanges config and sends message via channel", func(t *testing.T) {
		consumerChannelCreator := Manager{
			portPool:       port.NewPool(),
			broker:         mockBroker,
			consumerPinger: consumerPinger,
			pendingConfigs: map[PublicKey]*p2pConnectConfig{},
			signer:         signerFactory,
			verifier:       verifier,
			ipResolver:     ipResolver,
			brokerAddress:  "mock",
		}

		consumerChannel, err := consumerChannelCreator.CreateChannel(consumerID, providerID, 5*time.Second)
		assert.NoError(t, err)
		consumerChannel.SetSendTimeout(5 * time.Second)

		res, err := consumerChannel.Send("test", &Message{Data: []byte("ping")})
		assert.NoError(t, err)
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
