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
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/pb"

	nats_lib "github.com/nats-io/go-nats"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

// Listener knows how to exchange p2p keys and encrypted configuration and creates ready to use p2p channels.
type Listener interface {
	// Listen listens for incoming peer connections to establish new p2p channels. Establishes p2p channel and passes it
	// to channelHandlers
	Listen(providerID identity.Identity, serviceType string, channelHandler func(ch Channel)) error
}

// NewListener creates new p2p communication listener which is used on provider side.
func NewListener(broker brokerConnector, address string, signer identity.SignerFactory, verifier identity.Verifier, ipResolver ip.Resolver, providerPinger natProviderPinger, portPool port.ServicePortSupplier) Listener {
	return &listener{
		broker:         broker,
		brokerAddress:  address,
		pendingConfigs: map[PublicKey]*p2pConnectConfig{},
		ipResolver:     ipResolver,
		signer:         signer,
		verifier:       verifier,
		portPool:       portPool,
		providerPinger: providerPinger,
	}
}

// listener implements Listener interface.
type listener struct {
	portPool       port.ServicePortSupplier
	broker         brokerConnector
	providerPinger natProviderPinger
	signer         identity.SignerFactory
	verifier       identity.Verifier
	ipResolver     ip.Resolver
	brokerAddress  string

	// Keys holds pendingConfigs temporary configs for provider side since it
	// need to handle key exchange in two steps.
	pendingConfigs   map[PublicKey]*p2pConnectConfig
	pendingConfigsMu sync.Mutex
}

type p2pConnectConfig struct {
	publicIP     string
	peerPublicIP string
	peerPorts    []int
	localPorts   []int
	privateKey   PrivateKey
	peerPubKey   PublicKey
}

func (c *p2pConnectConfig) pingIP() string {
	if c.publicIP == c.peerPublicIP {
		// Assume that both peers are on the same network.
		return "127.0.0.1"
	}
	return c.peerPublicIP
}

// Listen listens for incoming peer connections to establish new p2p channels. Establishes p2p channel and passes it
// to channelHandlers
func (m *listener) Listen(providerID identity.Identity, serviceType string, channelHandlers func(ch Channel)) error {
	brokerConn, err := m.broker.Connect(m.brokerAddress)
	if err != nil {
		return err
	}
	// TODO: Expose func to close broker conn.

	outboundIP, err := m.ipResolver.GetOutboundIPAsString()
	if err != nil {
		return fmt.Errorf("could not get outbound IP: %w", err)
	}

	_, err = brokerConn.Subscribe(configExchangeSubject(providerID, serviceType), func(msg *nats_lib.Msg) {
		if err := m.providerStartConfigExchange(brokerConn, providerID, msg, outboundIP); err != nil {
			log.Err(err).Msg("Could not handle initial exchange")
			return
		}
	})

	_, err = brokerConn.Subscribe(configExchangeACKSubject(providerID, serviceType), func(msg *nats_lib.Msg) {
		config, err := m.providerAckConfigExchange(msg)
		if err != nil {
			log.Err(err).Msg("Could not handle exchange ack")
			return
		}

		// Send ack in separate goroutine and start pinging.
		// It is important that provider starts sending pings first otherwise
		// providers router can think that consumer is sending DDoS packets.
		go func(reply string) {
			if err := brokerConn.Publish(reply, []byte("OK")); err != nil {
				log.Err(err).Msg("Could not publish exchange ack")
			}
		}(msg.Reply)

		var conn1, conn2 *net.UDPConn
		if len(config.peerPorts) == requiredConnCount {
			log.Debug().Msg("Skipping consumer ping")
			conn1, err = net.DialUDP("udp4", &net.UDPAddr{Port: config.localPorts[0]}, &net.UDPAddr{IP: net.ParseIP(config.peerPublicIP), Port: config.peerPorts[0]})
			if err != nil {
				log.Err(err).Msg("Could not create UDP conn for p2p channel")
				return
			}
			conn2, err = net.DialUDP("udp4", &net.UDPAddr{Port: config.localPorts[1]}, &net.UDPAddr{IP: net.ParseIP(config.peerPublicIP), Port: config.peerPorts[1]})
			if err != nil {
				log.Err(err).Msg("Could not create UDP conn for service")
				return
			}
		} else {
			log.Debug().Msgf("Pinging consumer with IP %s using ports %v:%v", config.pingIP(), config.localPorts, config.peerPorts)
			conns, err := m.providerPinger.PingConsumerPeer(context.Background(), config.pingIP(), config.localPorts, config.peerPorts, providerInitialTTL, requiredConnCount)
			if err != nil {
				log.Err(err).Msg("Could not ping peer")
				return
			}
			conn1 = conns[0]
			conn2 = conns[1]
		}
		channel, err := newChannel(conn1, config.privateKey, config.peerPubKey)
		if err != nil {
			log.Err(err).Msg("Could not create channel")
			return
		}
		channel.setServiceConn(conn2)

		channelHandlers(channel)

		// Send handlers ready to consumer
		if err := m.providerChannelHandlersReady(brokerConn, providerID, serviceType); err != nil {
			log.Err(err).Msg("Could not handle channel handlers ready")
			return
		}
	})

	return err
}

func (m *listener) providerStartConfigExchange(brokerConn nats.Connection, signerID identity.Identity, msg *nats_lib.Msg, outboundIP string) error {
	pubKey, privateKey, err := GenerateKey()
	if err != nil {
		return fmt.Errorf("could not generate provider p2p keys: %w", err)
	}

	// Get initial peer exchange with it's public key.
	signedMsg, err := unpackSignedMsg(m.verifier, msg.Data)
	if err != nil {
		return fmt.Errorf("could not unpack signed msg: %w", err)
	}
	var peerExchangeMsg pb.P2PConfigExchangeMsg
	if err := proto.Unmarshal(signedMsg.Data, &peerExchangeMsg); err != nil {
		return err
	}
	peerPubKey, err := DecodePublicKey(peerExchangeMsg.PublicKey)
	if err != nil {
		return err
	}
	log.Debug().Msgf("Received consumer public key %s", peerPubKey.Hex())

	// Send reply with encrypted exchange config.
	publicIP, err := m.ipResolver.GetPublicIP()
	if err != nil {
		return fmt.Errorf("could not get public IP: %v", err)
	}
	portsCount := pingMaxPorts
	if outboundIP == publicIP {
		portsCount = requiredConnCount
	}
	localPorts, err := acquireLocalPorts(m.portPool, portsCount)
	if err != nil {
		return fmt.Errorf("could not acquire local ports: %v", err)
	}
	config := pb.P2PConnectConfig{
		PublicIP: publicIP,
		Ports:    intToInt32Slice(localPorts),
	}
	configCiphertext, err := encryptConnConfigMsg(&config, privateKey, peerPubKey)
	if err != nil {
		return fmt.Errorf("could not encrypt config msg: %v", err)
	}
	exchangeMsg := pb.P2PConfigExchangeMsg{
		PublicKey:        pubKey.Hex(),
		ConfigCiphertext: configCiphertext,
	}
	log.Debug().Msgf("Sending reply with public key %s and encrypted config to consumer", exchangeMsg.PublicKey)
	packedMsg, err := packSignedMsg(m.signer, signerID, &exchangeMsg)
	if err != nil {
		return fmt.Errorf("could not pack signed message: %v", err)
	}
	err = brokerConn.Publish(msg.Reply, packedMsg)
	if err != nil {
		return fmt.Errorf("could not publish message via broker: %v", err)
	}

	m.setPendingConfig(publicIP, peerPubKey, privateKey, localPorts)
	return nil
}

func (m *listener) providerAckConfigExchange(msg *nats_lib.Msg) (*p2pConnectConfig, error) {
	signedMsg, err := unpackSignedMsg(m.verifier, msg.Data)
	if err != nil {
		return nil, fmt.Errorf("could not unpack signed msg: %w", err)
	}
	var peerExchangeMsg pb.P2PConfigExchangeMsg
	if err := proto.Unmarshal(signedMsg.Data, &peerExchangeMsg); err != nil {
		return nil, fmt.Errorf("could not unmarshal exchange msg: %w", err)
	}
	peerPubKey, err := DecodePublicKey(peerExchangeMsg.PublicKey)
	if err != nil {
		return nil, err
	}

	defer m.deletePendingConfig(peerPubKey)
	config, ok := m.pendingConfig(peerPubKey)
	if !ok {
		return nil, fmt.Errorf("pending config not found for key %s", peerPubKey.Hex())
	}

	peerConfig, err := decryptConnConfigMsg(peerExchangeMsg.ConfigCiphertext, config.privateKey, peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt peer conn config: %w", err)
	}

	log.Debug().Msgf("Decrypted consumer config: %v", peerConfig)

	return &p2pConnectConfig{
		privateKey:   config.privateKey,
		localPorts:   config.localPorts,
		peerPubKey:   config.peerPubKey,
		peerPublicIP: peerConfig.PublicIP,
		peerPorts:    int32ToIntSlice(peerConfig.Ports),
	}, nil
}

func (m *listener) providerChannelHandlersReady(brokerConn nats.Connection, providerID identity.Identity, serviceType string) error {
	handlersReadyMsg := pb.P2PChannelHandlersReady{Value: "HANDLERS READY"}

	message, err := proto.Marshal(&handlersReadyMsg)
	if err != nil {
		return fmt.Errorf("could not marshal exchange msg: %w", err)
	}

	log.Debug().Msgf("Sending handlers ready message")
	return brokerConn.Publish(channelHandlersReadySubject(providerID, serviceType), message)
}

func (m *listener) pendingConfig(peerPubKey PublicKey) (*p2pConnectConfig, bool) {
	m.pendingConfigsMu.Lock()
	defer m.pendingConfigsMu.Unlock()
	config, ok := m.pendingConfigs[peerPubKey]
	return config, ok
}

func (m *listener) setPendingConfig(publicIP string, peerPubKey PublicKey, privateKey PrivateKey, localPorts []int) {
	m.pendingConfigsMu.Lock()
	defer m.pendingConfigsMu.Unlock()
	m.pendingConfigs[peerPubKey] = &p2pConnectConfig{
		publicIP:   publicIP,
		localPorts: localPorts,
		privateKey: privateKey,
		peerPubKey: peerPubKey,
	}
}

func (m *listener) deletePendingConfig(peerPubKey PublicKey) {
	m.pendingConfigsMu.Lock()
	defer m.pendingConfigsMu.Unlock()
	delete(m.pendingConfigs, peerPubKey)
}

func (m *listener) sendSignedMsg(brokerConn nats.Connection, subject string, msg []byte, timeout time.Duration) ([]byte, error) {
	reply, err := brokerConn.Request(subject, msg, timeout)
	if err != nil {
		return nil, fmt.Errorf("could send broker request to subject %s: %v", subject, err)
	}
	return reply.Data, nil
}
