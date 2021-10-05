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
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/nat/mapping"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/mysteriumnetwork/node/router"
	"github.com/mysteriumnetwork/node/trace"
)

func TestDialer_Exchange_And_Communication_With_Provider(t *testing.T) {
	providerPinger, consumerPinger := natTestPingers(t)

	tests := []struct {
		name              string
		ipResolver        ip.Resolver
		natProviderPinger natProviderPinger
		natConsumerPinger natConsumerPinger
		portMapper        mapping.PortMapper
	}{
		{
			name:              "Provider with public IP",
			ipResolver:        ip.NewResolverMock("127.0.0.1"),
			natProviderPinger: &mockProviderNATPinger{},
			natConsumerPinger: &mockConsumerNATPinger{},
			portMapper:        &mockPortMapper{},
		},
		{
			name:              "Provider behind NAT",
			ipResolver:        ip.NewResolverMockMultiple("127.0.0.1", "1.1.1.1"),
			natProviderPinger: providerPinger,
			natConsumerPinger: consumerPinger,
			portMapper:        &mockPortMapper{},
		},
		{
			name:              "Provider behind NAT with Upnp enabled",
			ipResolver:        ip.NewResolverMockMultiple("127.0.0.1", "1.1.1.1"),
			natProviderPinger: &mockProviderNATPinger{},
			natConsumerPinger: &mockConsumerNATPinger{},
			portMapper:        &mockPortMapper{enabled: true},
		},
		{
			name:              "Provider behind NAT with manual port forwarding and noop pinger",
			ipResolver:        ip.NewResolverMockMultiple("127.0.0.1", "1.1.1.1"),
			natProviderPinger: traversal.NewNoopPinger(eventbus.New()),
			natConsumerPinger: traversal.NewNoopPinger(eventbus.New()),
			portMapper:        &mockPortMapper{enabled: false},
		},
	}

	router.DefaultRouter = &mockRouter{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			providerID := identity.FromAddress("0x1")
			signerFactory := func(id identity.Identity) identity.Signer {
				return &identity.SignerFake{}
			}
			verifier := &identity.VerifierFake{}
			verifierFactory := func(_ identity.Identity) identity.Verifier {
				return verifier
			}
			brokerConn := nats.StartConnectionMock()
			defer brokerConn.Close()
			mockBroker := &mockBroker{conn: brokerConn}
			portPool := port.NewFixedRangePool(port.Range{Start: 10000, End: 60000})

			// Provider starts listening.
			channelListener := NewListener(brokerConn, signerFactory, verifier, test.ipResolver, eventbus.New())
			_, err := channelListener.Listen(providerID, "wireguard", func(ch Channel) {
				ch.Handle("test", func(c Context) error {
					return c.OkWithReply(&Message{Data: []byte("pong")})
				})
			})
			assert.NoError(t, err)

			// Consumer starts dialing provider.
			channelDialer := NewDialer(mockBroker, signerFactory, verifierFactory, test.ipResolver, portPool, nil)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			consumerChannel, err := channelDialer.Dial(ctx, identity.FromAddress("0x2"), providerID, "wireguard", ContactDefinition{BrokerAddresses: []string{"broker"}}, trace.NewTracer("Dial"))
			assert.NoError(t, err)
			defer consumerChannel.Close()

			res, err := consumerChannel.Send(context.Background(), "test", &Message{Data: []byte("ping")})
			assert.NoError(t, err)
			assert.Equal(t, "pong", string(res.Data))
		})
	}
}

func natTestPingers(t *testing.T) (providerPinger natProviderPinger, consumerPinger natConsumerPinger) {
	ports, err := acquirePorts(2)
	assert.NoError(t, err)
	providerPort := ports[0]
	consumerPort := ports[1]
	providerConn, err := net.DialUDP("udp", &net.UDPAddr{Port: providerPort}, &net.UDPAddr{Port: consumerPort})
	assert.NoError(t, err)
	consumerConn, err := net.DialUDP("udp", &net.UDPAddr{Port: consumerPort}, &net.UDPAddr{Port: providerPort})
	assert.NoError(t, err)
	providerPinger = &mockProviderNATPinger{conns: []*net.UDPConn{consumerConn, consumerConn}}
	consumerPinger = &mockConsumerNATPinger{conns: []*net.UDPConn{providerConn, providerConn}}
	return
}

type mockConsumerNATPinger struct {
	conns []*net.UDPConn
}

func (m *mockConsumerNATPinger) PingProviderPeer(ctx context.Context, localIP, remoteIP string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error) {
	return m.conns, nil
}

type mockProviderNATPinger struct {
	conns []*net.UDPConn
}

func (m *mockProviderNATPinger) PingConsumerPeer(ctx context.Context, id, ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error) {
	return m.conns, nil
}

type mockBroker struct {
	conn nats.Connection
}

func (m *mockBroker) Connect(serverURLs ...*url.URL) (nats.Connection, error) {
	return m.conn, nil
}

type mockPortMapper struct {
	enabled bool
}

func (m mockPortMapper) Map(id, protocol string, port int, name string) (release func(), ok bool) {
	return func() {}, m.enabled
}

type mockRouter struct{}

func (mr *mockRouter) ExcludeIP(net.IP) error        { return nil }
func (mr *mockRouter) RemoveExcludedIP(net.IP) error { return nil }
func (mr *mockRouter) Clean() error                  { return nil }
