package p2p

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/nat/traversal"
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

	ipResolver := ip.NewResolverMock("127.0.0.1")
	pinger := traversal.NewPinger(&traversal.PingConfig{
		Interval: 1 * time.Second,
		Timeout:  3 * time.Second,
	}, eventbus.New())

	t.Run("Test provider subscribes to channels requests", func(t *testing.T) {
		providerChannelSubscriber := Manager{
			portPool:       port.NewPool(),
			broker:         mockBroker,
			pinger:         pinger,
			pendingConfigs: map[PublicKey]*p2pConnectConfig{},
			signer:         signerFactory,
			verifier:       verifier,
			ipResolver:     ipResolver,
			brokerAddress:  "mock",
			log: func(msg ...interface{}) {
				fmt.Println("PROVIDER", msg)
			},
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
			pinger:         pinger,
			pendingConfigs: map[PublicKey]*p2pConnectConfig{},
			signer:         signerFactory,
			verifier:       verifier,
			ipResolver:     ipResolver,
			brokerAddress:  "mock",
			log: func(msg ...interface{}) {
				fmt.Println("CONSUMER", msg)
			},
		}

		consumerChannel, err := consumerChannelCreator.CreateChannel(consumerID, providerID, 5*time.Second)
		assert.NoError(t, err)
		consumerChannel.SetSendTimeout(5 * time.Second)

		res, err := consumerChannel.Send("test", &Message{Data: []byte("ping")})
		assert.NoError(t, err)
		assert.Equal(t, "pong", string(res.Data))
	})
}

type mockBroker struct {
	conn nats.Connection
}

func (m *mockBroker) Connect(serverURIs ...string) (nats.Connection, error) {
	return m.conn, nil
}

type mockPinger struct {
}

func (m *mockPinger) PingPeer(ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error) {
	return nil, nil
}
