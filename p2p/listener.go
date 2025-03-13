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

	nats_lib "github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/mysteriumnetwork/node/p2p/compat"
	"github.com/mysteriumnetwork/node/p2p/nat"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/services/quic/service/client"
	"github.com/mysteriumnetwork/node/trace"
)

// Listener knows how to exchange p2p keys and encrypted configuration and creates ready to use p2p channels.
type Listener interface {
	// Listen listens for incoming peer connections to establish new p2p channels. Establishes p2p channel and passes it
	// to channelHandlers
	Listen(providerID identity.Identity, serviceType string, channelHandler func(ch Channel)) (func(), error)

	// GetContact returns contract which is later can be added to proposal contacts definition so consumer can
	// know how to connect to this p2p listener.
	GetContact() market.Contact
}

// NewListener creates new p2p communication listener which is used on provider side.
func NewListener(brokerConn nats.Connection, signer identity.SignerFactory, verifier identity.Verifier, ipResolver ip.Resolver, eventBus eventbus.EventBus) Listener {
	return &listener{
		brokerConn:     brokerConn,
		pendingConfigs: map[PublicKey]p2pConnectConfig{},
		ipResolver:     ipResolver,
		signer:         signer,
		verifier:       verifier,
		eventBus:       eventBus,
	}
}

// listener implements Listener interface.
type listener struct {
	eventBus   eventbus.EventBus
	brokerConn nats.Connection
	signer     identity.SignerFactory
	verifier   identity.Verifier
	ipResolver ip.Resolver

	// Keys holds pendingConfigs temporary configs for provider side since it
	// need to handle key exchange in two steps.
	pendingConfigs   map[PublicKey]p2pConnectConfig
	pendingConfigsMu sync.Mutex
}

type p2pConnectConfig struct {
	publicIP         string
	peerPublicIP     string
	peerURL          string
	compatibility    int
	peerPorts        []int
	localPorts       []int
	publicPorts      []int
	publicKey        PublicKey
	privateKey       PrivateKey
	peerPubKey       PublicKey
	tracer           *trace.Tracer
	upnpPortsRelease func()
	start            nat.StartPorts
	peerID           identity.Identity
}

func (c *p2pConnectConfig) peerIP() string {
	if c.publicIP == c.peerPublicIP {
		// Assume that both peers are on the same network.
		return "127.0.0.1"
	}
	return c.peerPublicIP
}

func (m *listener) GetContact() market.Contact {
	return market.Contact{
		Type:       ContactTypeV1,
		Definition: ContactDefinition{BrokerAddresses: m.brokerConn.Servers()},
	}
}

// Listen listens for incoming peer connections to establish new p2p channels. Establishes p2p channel and passes it
// to channelHandlers.
func (m *listener) Listen(providerID identity.Identity, serviceType string, channelHandlers func(ch Channel)) (func(), error) {
	configSignedSubject, err := nats.SignedSubject(m.signer(providerID), configExchangeSubject(providerID, serviceType))
	if err != nil {
		return func() {}, fmt.Errorf("cannot sign config topic: %w", err)
	}

	configSub, err := m.brokerConn.Subscribe(configSignedSubject, func(msg *nats_lib.Msg) {
		if err := m.providerStartConfigExchange(providerID, serviceType, msg); err != nil {
			log.Err(err).Msg("Could not handle initial exchange")
			return
		}
	})
	if err != nil {
		return func() {}, fmt.Errorf("could not get subscribe to config exchange topic: %w", err)
	}

	ackSignedSubject, err := nats.SignedSubject(m.signer(providerID), configExchangeACKSubject(providerID, serviceType))
	if err != nil {
		return func() {}, fmt.Errorf("cannot sign ack topic: %w", err)
	}

	ackSub, err := m.brokerConn.Subscribe(ackSignedSubject, func(msg *nats_lib.Msg) {
		config, err := m.providerAckConfigExchange(msg)
		if err != nil {
			log.Err(err).Msg("Could not handle exchange ack")
			return
		}

		trace := config.tracer.StartStage("Provider P2P exchange ack")
		// Send ack in separate goroutine and start pinging.
		// It is important that provider starts sending pings first otherwise
		// providers router can think that consumer is sending DDoS packets.
		go func(reply string) {
			// race condition still happens when consumer starts to ping until provider did not manage to complete required number of pings
			// this might be provider / consumer performance dependent
			// make sleep time dependent on pinger interval and wait for 2 ping iterations
			// TODO: either reintroduce eventual increase of TTL on consumer or maintain some sane delay
			dur := traversal.DefaultPingConfig().Interval.Milliseconds() * int64(len(config.localPorts)) / 2
			log.Debug().Msgf("Delaying pings from consumer for %v ms", dur)
			time.Sleep(time.Duration(dur) * time.Millisecond)

			if err := m.brokerConn.Publish(reply, []byte("OK")); err != nil {
				log.Err(err).Msg("Could not publish exchange ack")
			}
			config.tracer.EndStage(trace)
		}(msg.Reply)

		ctx, cancel := context.WithCancel(context.Background())

		var conn1, conn2 ServiceConn
		if config.start != nil {
			traceDial := config.tracer.StartStage("Provider P2P dial (preparation)")
			log.Debug().Msgf("Pinging consumer using ports %v:%v initial ttl: %v", config.localPorts, config.peerPorts, 1)

			conns, err := config.start(ctx, config.peerIP(), config.peerPorts, config.localPorts)
			if err != nil {
				log.Err(err).Msg("Could not ping peer")
				return
			}

			if len(conns) != requiredConnCount {
				log.Err(err).Msg("Could not get required number of connections")
				return
			}

			conn1 = conns[0]
			conn2 = conns[1]
			config.tracer.EndStage(traceDial)
		} else {
			traceDial := config.tracer.StartStage("Provider P2P dial (direct)")
			log.Debug().Msg("Skipping consumer ping")

			if config.peerURL == "" {
				conn1, err = net.DialUDP("udp4", &net.UDPAddr{Port: config.localPorts[0]}, &net.UDPAddr{IP: net.ParseIP(config.peerIP()), Port: config.peerPorts[0]})
				if err != nil {
					log.Err(err).Msg("Could not create UDP conn for p2p channel")
					return
				}
				conn2, err = net.DialUDP("udp4", &net.UDPAddr{Port: config.localPorts[1]}, &net.UDPAddr{IP: net.ParseIP(config.peerIP()), Port: config.peerPorts[1]})
				if err != nil {
					log.Err(err).Msg("Could not create UDP conn for service")
					return
				}
			} else {
				log.Info().Str("url", config.peerURL).Msg("Consumer provided URL, skipping p2p connection establishment")

				client := client.NewClient(config.peerURL)
				conn1, err = client.DialCommunication(ctx)
				if err != nil {
					log.Err(err).Msg("Could not dial myst-communication")
					return
				}

				conn2, err = client.DialTransport(ctx)
				if err != nil {
					log.Err(err).Msg("Could not dial myst-transport")
					return
				}
			}

			config.tracer.EndStage(traceDial)
		}

		traceAck := config.tracer.StartStage("Provider P2P dial ack")

		var channel communicationChannel
		if serviceType == "quic_scraping" {
			channel = newChannelQuic(conn1, config.peerID, config.compatibility)
		} else {
			channel, err = newChannel(conn1, config.privateKey, config.peerPubKey, config.compatibility)
		}

		if err != nil {
			log.Err(err).Msg("Could not create channel")
			return
		}
		channel.setTracer(config.tracer)
		channel.setServiceConn(conn2)
		channel.setPeerID(config.peerID)
		channel.setUpnpPortsRelease(func() {
			cancel()
			if config.upnpPortsRelease != nil {
				config.upnpPortsRelease()
			}
		})

		channelHandlers(channel)

		channel.launchReadSendLoops()

		// Send handlers ready to consumer.
		if err := m.providerChannelHandlersReady(providerID, serviceType); err != nil {
			log.Err(err).Msg("Could not handle channel handlers ready")
			channel.Close()
			return
		}
		config.tracer.EndStage(traceAck)
	})
	if err != nil {
		if err := configSub.Unsubscribe(); err != nil {
			log.Err(err).Msg("Failed to unsubscribe from config exchange topic")
		}

		return func() {}, fmt.Errorf("could not get subscribe to config exchange acknowledge topic: %w", err)
	}

	return func() {
		if err := configSub.Unsubscribe(); err != nil {
			log.Err(err).Msg("Failed to unsubscribe from config exchange topic")
		}
		if err := ackSub.Unsubscribe(); err != nil {
			log.Err(err).Msg("Failed to unsubscribe from config exchange acknowledge topic")
		}
	}, nil
}

func (m *listener) providerStartConfigExchange(providerID identity.Identity, serviceType string, msg *nats_lib.Msg) error {
	tracer := trace.NewTracer("Provider whole Connect")

	trace := tracer.StartStage("Provider P2P exchange")
	defer tracer.EndStage(trace)

	pubKey, privateKey, err := GenerateKey()
	if err != nil {
		return fmt.Errorf("could not generate provider p2p keys: %w", err)
	}

	// Get initial peer exchange with it's public key.
	signedMsg, peerID, err := unpackSignedMsg(m.verifier, msg.Data)
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

	p2pConnConfig := p2pConnectConfig{
		publicKey:    pubKey,
		privateKey:   privateKey,
		peerPubKey:   peerPubKey,
		tracer:       tracer,
		peerPublicIP: "",
		peerPorts:    nil,
		peerID:       peerID,
	}

	config := pb.P2PConnectConfig{
		Ports:         intToInt32Slice(p2pConnConfig.publicPorts),
		Compatibility: compat.Compatibility,
	}

	if serviceType != "quic_scraping" {
		publicIP, localPorts, portsRelease, start, err := m.prepareLocalPorts(providerID.Address, tracer)
		if err != nil {
			return fmt.Errorf("could not prepare ports: %w", err)
		}

		p2pConnConfig.publicIP = publicIP
		p2pConnConfig.localPorts = localPorts
		p2pConnConfig.publicPorts = stunPorts(providerID, m.eventBus, localPorts...)
		p2pConnConfig.upnpPortsRelease = portsRelease
		p2pConnConfig.start = start

		config.PublicIP = publicIP
		config.Ports = intToInt32Slice(p2pConnConfig.publicPorts)
	}

	m.setPendingConfig(p2pConnConfig)

	configCiphertext, err := encryptConnConfigMsg(&config, privateKey, peerPubKey)
	if err != nil {
		return fmt.Errorf("could not encrypt config msg: %w", err)
	}
	exchangeMsg := pb.P2PConfigExchangeMsg{
		PublicKey:        pubKey.Hex(),
		ConfigCiphertext: configCiphertext,
	}
	log.Debug().Msgf("Sending reply with public key %s and encrypted config to consumer", exchangeMsg.PublicKey)
	packedMsg, err := packSignedMsg(m.signer, providerID, &exchangeMsg)
	if err != nil {
		return fmt.Errorf("could not pack signed message: %w", err)
	}
	err = m.brokerConn.Publish(msg.Reply, packedMsg)
	if err != nil {
		return fmt.Errorf("could not publish message via broker: %w", err)
	}
	return nil
}

// prepareLocalPorts acquires ports for p2p connections. It tries to acquire only
// required ports count for actual p2p and service connections and fallback to
// acquiring extra ports for nat pinger if provider is behind nat, port mapping failed
// and no manual port forwarding is enabled.
func (m *listener) prepareLocalPorts(id string, tracer *trace.Tracer) (string, []int, func(), nat.StartPorts, error) {
	trace := tracer.StartStage("Provider P2P exchange (ports)")
	defer tracer.EndStage(trace)

	publicIP, err := m.ipResolver.GetPublicIP()
	if err != nil {
		return "", nil, nil, nil, fmt.Errorf("could not get public IP: %w", err)
	}

	for _, p := range nat.OrderedPortProviders() {
		ports, release, start, err := p.Provider.PreparePorts()
		if err == nil {
			m.eventBus.Publish(nat.AppTopicNATTraversalMethod, nat.NATTraversalMethod{
				Identity: id,
				Method:   p.Method,
				Success:  true,
			})
			return publicIP, ports, release, start, nil
		}

		m.eventBus.Publish(nat.AppTopicNATTraversalMethod, nat.NATTraversalMethod{
			Identity: id,
			Method:   p.Method,
			Success:  false,
		})
	}

	return "", nil, nil, nil, fmt.Errorf("failed to prepare local ports")
}

func (m *listener) providerAckConfigExchange(msg *nats_lib.Msg) (*p2pConnectConfig, error) {
	signedMsg, peerID, err := unpackSignedMsg(m.verifier, msg.Data)
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
	if peerID != config.peerID {
		return nil, fmt.Errorf("acknowledged config signed by unexpected identity: %s", peerID.ToCommonAddress())
	}

	peerConfig, err := decryptConnConfigMsg(peerExchangeMsg.ConfigCiphertext, config.privateKey, peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt peer conn config: %w", err)
	}

	return &p2pConnectConfig{
		peerPublicIP:     peerConfig.GetPublicIP(),
		peerPorts:        int32ToIntSlice(peerConfig.GetPorts()),
		peerURL:          peerConfig.GetUrl(),
		compatibility:    int(peerConfig.GetCompatibility()),
		localPorts:       config.localPorts,
		publicKey:        config.publicKey,
		privateKey:       config.privateKey,
		peerPubKey:       config.peerPubKey,
		publicIP:         config.publicIP,
		tracer:           config.tracer,
		upnpPortsRelease: config.upnpPortsRelease,
		start:            config.start,
		peerID:           config.peerID,
	}, nil
}

func (m *listener) providerChannelHandlersReady(providerID identity.Identity, serviceType string) error {
	handlersReadyMsg := pb.P2PChannelHandlersReady{Value: "HANDLERS READY"}

	message, err := proto.Marshal(&handlersReadyMsg)
	if err != nil {
		return fmt.Errorf("could not marshal exchange msg: %w", err)
	}

	signedSubject, err := nats.SignedSubject(m.signer(providerID), channelHandlersReadySubject(providerID, serviceType))
	if err != nil {
		return fmt.Errorf("unable to sign p2p-channel-handlers-ready subject: %w", err)
	}

	log.Debug().Msgf("Sending handlers ready message")
	return m.brokerConn.Publish(signedSubject, message)
}

func (m *listener) pendingConfig(peerPubKey PublicKey) (p2pConnectConfig, bool) {
	m.pendingConfigsMu.Lock()
	defer m.pendingConfigsMu.Unlock()
	config, ok := m.pendingConfigs[peerPubKey]
	return config, ok
}

func (m *listener) setPendingConfig(config p2pConnectConfig) {
	m.pendingConfigsMu.Lock()
	defer m.pendingConfigsMu.Unlock()
	m.pendingConfigs[config.peerPubKey] = config
}

func (m *listener) deletePendingConfig(peerPubKey PublicKey) {
	m.pendingConfigsMu.Lock()
	defer m.pendingConfigsMu.Unlock()
	delete(m.pendingConfigs, peerPubKey)
}
