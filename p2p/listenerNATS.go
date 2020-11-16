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
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat/mapping"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/trace"

	nats_lib "github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

// NewListenerNATS creates new p2p communication listener which is used on provider side.
func NewListenerNATS(brokerConn nats.Connection, signer identity.SignerFactory, verifier identity.Verifier, ipResolver ip.Resolver, providerPinger natProviderPinger, portPool port.ServicePortSupplier, portMapper mapping.PortMapper) Listener {
	return &listenerNATS{
		brokerConn:     brokerConn,
		pendingConfigs: map[PublicKey]p2pConnectConfig{},
		ipResolver:     ipResolver,
		signer:         signer,
		verifier:       verifier,
		portPool:       portPool,
		providerPinger: providerPinger,
		portMapper:     portMapper,
	}
}

// listenerNATS implements Listener interface.
type listenerNATS struct {
	portPool       port.ServicePortSupplier
	brokerConn     nats.Connection
	providerPinger natProviderPinger
	signer         identity.SignerFactory
	verifier       identity.Verifier
	ipResolver     ip.Resolver
	portMapper     mapping.PortMapper

	// Keys holds pendingConfigs temporary configs for provider side since it
	// need to handle key exchange in two steps.
	pendingConfigs   map[PublicKey]p2pConnectConfig
	pendingConfigsMu sync.Mutex
}

type p2pConnectConfig struct {
	publicIP         string
	peerPublicIP     string
	peerPorts        []int
	localPorts       []int
	publicKey        PublicKey
	privateKey       PrivateKey
	peerPubKey       PublicKey
	tracer           *trace.Tracer
	upnpPortsRelease []func()
}

func (c *p2pConnectConfig) peerIP() string {
	if c.publicIP == c.peerPublicIP {
		// Assume that both peers are on the same network.
		return "127.0.0.1"
	}
	return c.peerPublicIP
}

func (l *listenerNATS) GetContacts(_, _ string) []market.Contact {
	return []market.Contact{{
		Type:       ContactTypeNATSv1,
		Definition: ContactDefinition{BrokerAddresses: l.brokerConn.Servers()}}}
}

// Listen listens for incoming peer connections to establish new p2p channels. Establishes p2p channel and passes it
// to channelHandlers.
func (l *listenerNATS) Listen(providerID identity.Identity, serviceType string, channelHandlers func(ch Channel)) (func(), error) {
	outboundIP, err := l.ipResolver.GetOutboundIP()
	if err != nil {
		return func() {}, fmt.Errorf("could not get outbound IP: %w", err)
	}

	configSub, err := l.brokerConn.Subscribe(configExchangeSubject(providerID, serviceType), func(msg *nats_lib.Msg) {
		if err := l.providerStartConfigExchange(providerID, msg, outboundIP); err != nil {
			log.Err(err).Msg("Could not handle initial exchange")
			return
		}
	})
	if err != nil {
		return func() {}, fmt.Errorf("could not get subscribe to config exchange topic: %w", err)
	}

	ackSub, err := l.brokerConn.Subscribe(configExchangeACKSubject(providerID, serviceType), func(msg *nats_lib.Msg) {
		config, err := l.providerAckConfigExchange(msg)
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

			if err := l.brokerConn.Publish(reply, []byte("OK")); err != nil {
				log.Err(err).Msg("Could not publish exchange ack")
			}
			config.tracer.EndStage(trace)
		}(msg.Reply)

		var conn1, conn2 *net.UDPConn
		if len(config.peerPorts) == requiredConnCount {
			traceDial := config.tracer.StartStage("Provider P2P dial (upnp)")
			log.Debug().Msg("Skipping consumer ping")
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
			config.tracer.EndStage(traceDial)
		} else {
			traceDial := config.tracer.StartStage("Provider P2P dial (pinger)")
			log.Debug().Msgf("Pinging consumer with IP %s using ports %v:%v initial ttl: %v",
				config.peerIP(), config.localPorts, config.peerPorts, providerInitialTTL)
			conns, err := l.providerPinger.PingConsumerPeer(context.Background(), config.peerIP(), config.localPorts, config.peerPorts, providerInitialTTL, requiredConnCount)
			if err != nil {
				log.Err(err).Msg("Could not ping peer")
				return
			}
			conn1 = conns[0]
			conn2 = conns[1]
			config.tracer.EndStage(traceDial)
		}

		traceAck := config.tracer.StartStage("Provider P2P dial ack")
		channel, err := newChannel(conn1, config.privateKey, config.peerPubKey)
		if err != nil {
			log.Err(err).Msg("Could not create channel")
			return
		}
		channel.setTracer(config.tracer)
		channel.setServiceConn(conn2)
		channel.setUpnpPortsRelease(config.upnpPortsRelease)

		channelHandlers(channel)

		channel.launchReadSendLoops()

		// Send handlers ready to consumer.
		if err := l.providerChannelHandlersReady(providerID, serviceType); err != nil {
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

func (l *listenerNATS) providerStartConfigExchange(signerID identity.Identity, msg *nats_lib.Msg, outboundIP string) error {
	tracer := trace.NewTracer("Provider whole Connect")

	trace := tracer.StartStage("Provider P2P exchange")
	defer tracer.EndStage(trace)

	pubKey, privateKey, err := GenerateKey()
	if err != nil {
		return fmt.Errorf("could not generate provider p2p keys: %w", err)
	}

	// Get initial peer exchange with it's public key.
	signedMsg, err := unpackSignedMsg(l.verifier, msg.Data)
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

	publicIP, localPorts, portsRelease, err := l.prepareLocalPorts(outboundIP, tracer)
	if err != nil {
		return fmt.Errorf("could not prepare ports: %w", err)
	}

	l.setPendingConfig(p2pConnectConfig{
		publicIP:         publicIP,
		localPorts:       localPorts,
		publicKey:        pubKey,
		privateKey:       privateKey,
		peerPubKey:       peerPubKey,
		tracer:           tracer,
		upnpPortsRelease: portsRelease,
		peerPublicIP:     "",
		peerPorts:        nil,
	})

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
	packedMsg, err := packSignedMsg(l.signer, signerID, &exchangeMsg)
	if err != nil {
		return fmt.Errorf("could not pack signed message: %v", err)
	}
	err = l.brokerConn.Publish(msg.Reply, packedMsg)
	if err != nil {
		return fmt.Errorf("could not publish message via broker: %v", err)
	}
	return nil
}

// prepareLocalPorts acquires ports for p2p connections. It tries to acquire only
// required ports count for actual p2p and service connections and fallback to
// acquiring extra ports for nat pinger if provider is behind nat, port mapping failed
// and no manual port forwarding is enabled.
func (l *listenerNATS) prepareLocalPorts(outboundIP string, tracer *trace.Tracer) (string, []int, []func(), error) {
	trace := tracer.StartStage("Provider P2P exchange (ports)")
	defer tracer.EndStage(trace)

	// Send reply with encrypted exchange config.
	publicIP, err := l.ipResolver.GetPublicIP()
	if err != nil {
		return "", nil, nil, fmt.Errorf("could not get public IP: %v", err)
	}

	// First acquire required only ports for needed n connections.
	localPorts, err := acquireLocalPorts(l.portPool, requiredConnCount)
	if err != nil {
		return "", nil, nil, fmt.Errorf("could not acquire initial local ports: %v", err)
	}
	// Return these ports if provider is not behind NAT.
	if outboundIP == publicIP {
		return publicIP, localPorts, nil, nil
	}

	// Try to add upnp ports mapping.
	var portsRelease []func()
	var portMappingOk bool
	var portRelease func()
	for _, p := range localPorts {
		portRelease, portMappingOk = l.portMapper.Map("UDP", p, "Myst node p2p port mapping")
		if !portMappingOk {
			break
		}
		portsRelease = append(portsRelease, portRelease)
	}
	if portMappingOk {
		return publicIP, localPorts, portsRelease, nil
	}

	// Check if nat pinger is valid. It's considered as not valid when noop pinger is used in case
	// manual port forwarding is specified.
	if _, noop := l.providerPinger.(*traversal.NoopPinger); noop {
		return publicIP, localPorts, nil, nil
	}

	// Acquire more ports for nat pinger.
	morePorts, err := acquireLocalPorts(l.portPool, pingMaxPorts-requiredConnCount)
	if err != nil {
		return publicIP, nil, nil, fmt.Errorf("could not acquire more local ports: %v", err)
	}
	for _, p := range morePorts {
		localPorts = append(localPorts, p)
	}
	return publicIP, localPorts, nil, nil
}

func (l *listenerNATS) providerAckConfigExchange(msg *nats_lib.Msg) (*p2pConnectConfig, error) {
	signedMsg, err := unpackSignedMsg(l.verifier, msg.Data)
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

	defer l.deletePendingConfig(peerPubKey)
	config, ok := l.pendingConfig(peerPubKey)
	if !ok {
		return nil, fmt.Errorf("pending config not found for key %s", peerPubKey.Hex())
	}

	peerConfig, err := decryptConnConfigMsg(peerExchangeMsg.ConfigCiphertext, config.privateKey, peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt peer conn config: %w", err)
	}

	log.Debug().Msgf("Decrypted consumer config: %v", peerConfig)

	return &p2pConnectConfig{
		peerPublicIP:     peerConfig.PublicIP,
		peerPorts:        int32ToIntSlice(peerConfig.Ports),
		localPorts:       config.localPorts,
		publicKey:        config.publicKey,
		privateKey:       config.privateKey,
		peerPubKey:       config.peerPubKey,
		publicIP:         config.publicIP,
		tracer:           config.tracer,
		upnpPortsRelease: config.upnpPortsRelease,
	}, nil
}

func (l *listenerNATS) providerChannelHandlersReady(providerID identity.Identity, serviceType string) error {
	handlersReadyMsg := pb.P2PChannelHandlersReady{Value: "HANDLERS READY"}

	message, err := proto.Marshal(&handlersReadyMsg)
	if err != nil {
		return fmt.Errorf("could not marshal exchange msg: %w", err)
	}

	log.Debug().Msgf("Sending handlers ready message")
	return l.brokerConn.Publish(channelHandlersReadySubject(providerID, serviceType), message)
}

func (l *listenerNATS) pendingConfig(peerPubKey PublicKey) (p2pConnectConfig, bool) {
	l.pendingConfigsMu.Lock()
	defer l.pendingConfigsMu.Unlock()
	config, ok := l.pendingConfigs[peerPubKey]
	return config, ok
}

func (l *listenerNATS) setPendingConfig(config p2pConnectConfig) {
	l.pendingConfigsMu.Lock()
	defer l.pendingConfigsMu.Unlock()
	l.pendingConfigs[config.peerPubKey] = config
}

func (l *listenerNATS) deletePendingConfig(peerPubKey PublicKey) {
	l.pendingConfigsMu.Lock()
	defer l.pendingConfigsMu.Unlock()
	delete(l.pendingConfigs, peerPubKey)
}
