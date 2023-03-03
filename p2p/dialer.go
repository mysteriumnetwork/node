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
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	nats_lib "github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/mysteriumnetwork/node/p2p/compat"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/router"
	"github.com/mysteriumnetwork/node/trace"
)

const maxBrokerConnectAttempts = 25

// Dialer knows how to exchange p2p keys and encrypted configuration and creates ready to use p2p channels.
type Dialer interface {
	// Dial exchanges p2p configuration via broker, performs NAT pinging if needed
	// and create p2p channel which is ready for communication.
	Dial(ctx context.Context, consumerID, providerID identity.Identity, serviceType string, contactDef ContactDefinition, tracer *trace.Tracer) (Channel, error)
}

// NewDialer creates new p2p communication dialer which is used on consumer side.
func NewDialer(broker brokerConnector, signer identity.SignerFactory, verifierFactory identity.VerifierFactory, ipResolver ip.Resolver, portPool port.ServicePortSupplier, eventBus eventbus.EventBus) Dialer {
	return &dialer{
		broker:          broker,
		ipResolver:      ipResolver,
		signer:          signer,
		verifierFactory: verifierFactory,
		portPool:        portPool,
		consumerPinger:  traversal.NewPinger(traversal.DefaultPingConfig(), eventbus.New()),
		eventBus:        eventBus,
	}
}

// dialer implements Dialer interface.
type dialer struct {
	portPool        port.ServicePortSupplier
	broker          brokerConnector
	consumerPinger  natConsumerPinger
	signer          identity.SignerFactory
	verifierFactory identity.VerifierFactory
	ipResolver      ip.Resolver
	eventBus        eventbus.EventBus
}

// Dial exchanges p2p configuration via broker, performs NAT pinging if needed
// and create p2p channel which is ready for communication.
func (m *dialer) Dial(ctx context.Context, consumerID, providerID identity.Identity, serviceType string, contactDef ContactDefinition, tracer *trace.Tracer) (Channel, error) {
	config := &p2pConnectConfig{tracer: tracer}

	// Send initial exchange with signed consumer public key.
	brokerConn, err := m.connect(contactDef, tracer)
	if err != nil {
		return nil, fmt.Errorf("could not open broker conn: %w", err)
	}
	defer brokerConn.Close()

	peerReady := make(chan struct{})
	var once sync.Once
	_, err = brokerConn.Subscribe(channelHandlersReadySubject(providerID, serviceType), func(msg *nats_lib.Msg) {
		defer once.Do(func() { close(peerReady) })
		if err := m.channelHandlersReady(msg); err != nil {
			log.Err(err).Msg("Channel handlers ready handler setup failed")
			return
		}
	})
	if err != nil {
		return nil, fmt.Errorf("could not subscribe to ready subject: %w", err)
	}

	config, err = m.startConfigExchange(config, ctx, brokerConn, providerID, serviceType, consumerID)
	if err != nil {
		return nil, fmt.Errorf("could not exchange config: %w", err)
	}

	if config.compatibility < 2 {
		return nil, fmt.Errorf("peer using compatibility version lower than 2: %d", config.compatibility)
	}

	if serviceType != "openvpn" { // OpenVPN does this automatically, we don't need to perform it manually.
		if err := router.ExcludeIP(net.ParseIP(config.peerIP())); err != nil {
			return nil, fmt.Errorf("failed to exclude peer IP from default routes: %w", err)
		}
	}

	if _, err := firewall.AllowIPAccess(config.peerPublicIP); err != nil {
		return nil, fmt.Errorf("could not add peer IP firewall rule: %w", err)
	}

	config.publicIP, config.localPorts, err = m.prepareLocalPorts(config)
	if err != nil {
		return nil, fmt.Errorf("could not prepare ports: %w", err)
	}

	config.publicPorts = stunPorts(consumerID, m.eventBus, config.localPorts...)

	// Finally send consumer encrypted and signed connect config in ack message.
	err = m.ackConfigExchange(config, ctx, brokerConn, providerID, serviceType, consumerID)
	if err != nil {
		return nil, fmt.Errorf("could not ack config: %w", err)
	}

	dial := m.dialPinger
	if len(config.peerPorts) == requiredConnCount {
		dial = m.dialDirect
	}
	conn1, conn2, err := dial(ctx, providerID, config)
	if err != nil {
		return nil, fmt.Errorf("could not dial p2p channel: %w", err)
	}

	// Wait until provider confirms that channel handlers are ready.
	traceAck := config.tracer.StartStage("Consumer P2P dial ack")
	select {
	case <-peerReady:
		log.Debug().Msg("Received handlers ready message from provider")
	case <-ctx.Done():
		return nil, errors.New("timeout while performing configuration exchange")
	}

	channel, err := newChannel(conn1, config.privateKey, config.peerPubKey, config.compatibility)
	if err != nil {
		return nil, fmt.Errorf("could not create p2p channel during dial: %w", err)
	}
	channel.setTracer(tracer)
	channel.setServiceConn(conn2)
	channel.setPeerID(providerID)
	channel.launchReadSendLoops()
	config.tracer.EndStage(traceAck)

	return channel, nil
}

func (m *dialer) connect(contactDef ContactDefinition, tracer *trace.Tracer) (conn nats.Connection, err error) {
	trace := tracer.StartStage("Consumer P2P connect")
	defer tracer.EndStage(trace)

	// broker connect might fail due to reconfiguration of network routes in progress
	for i := 0; i < maxBrokerConnectAttempts; i++ {
		serverURLs, err := nats.ParseServerURIs(contactDef.BrokerAddresses)
		if err != nil {
			return nil, err
		}

		conn, err = m.broker.Connect(serverURLs...)
		if err != nil {
			log.Warn().Msgf("broker connect failed - attempting again in 1sec: %s", err)
			time.Sleep(time.Second)
			continue
		}
		break
	}
	return conn, err
}

func (m *dialer) startConfigExchange(config *p2pConnectConfig, ctx context.Context, brokerConn nats.Connection, providerID identity.Identity, serviceType string, consumerID identity.Identity) (*p2pConnectConfig, error) {
	trace := config.tracer.StartStage("Consumer P2P exchange")
	defer config.tracer.EndStage(trace)

	pubKey, privateKey, err := GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("could not generate consumer p2p keys: %w", err)
	}

	beginExchangeMsg := &pb.P2PConfigExchangeMsg{
		PublicKey: pubKey.Hex(),
	}
	log.Debug().Msgf("Consumer %s sending public key %s to provider %s", consumerID.Address, beginExchangeMsg.PublicKey, providerID.Address)
	packedMsg, err := packSignedMsg(m.signer, consumerID, beginExchangeMsg)
	if err != nil {
		return nil, fmt.Errorf("could not pack signed message: %v", err)
	}
	exchangeMsgBrokerReply, err := m.sendSignedMsg(ctx, configExchangeSubject(providerID, serviceType), packedMsg, brokerConn)
	if err != nil {
		return nil, fmt.Errorf("could not send signed message: %w", err)
	}

	// Parse provider response with public key and encrypted and signed connection config.
	exchangeMsgReplySignedMsg, _, err := unpackSignedMsg(m.verifierFactory(providerID), exchangeMsgBrokerReply)
	if err != nil {
		return nil, fmt.Errorf("could not unpack peer signed message: %w", err)
	}
	var exchangeMsgReply pb.P2PConfigExchangeMsg
	if err := proto.Unmarshal(exchangeMsgReplySignedMsg.Data, &exchangeMsgReply); err != nil {
		return nil, fmt.Errorf("could not unmarshal peer signed message payload: %w", err)
	}
	peerPubKey, err := DecodePublicKey(exchangeMsgReply.PublicKey)
	if err != nil {
		return nil, err
	}
	peerConnConfig, err := decryptConnConfigMsg(exchangeMsgReply.ConfigCiphertext, privateKey, peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt peer conn config: %w", err)
	}

	config.publicKey = pubKey
	config.compatibility = int(peerConnConfig.Compatibility)
	config.privateKey = privateKey
	config.peerPubKey = peerPubKey
	config.peerPublicIP = peerConnConfig.PublicIP
	config.peerPorts = int32ToIntSlice(peerConnConfig.Ports)
	return config, nil
}

func (m *dialer) ackConfigExchange(config *p2pConnectConfig, ctx context.Context, brokerConn nats.Connection, providerID identity.Identity, serviceType string, consumerID identity.Identity) error {
	trace := config.tracer.StartStage("Consumer P2P exchange ack")
	defer config.tracer.EndStage(trace)

	connConfig := &pb.P2PConnectConfig{
		PublicIP:      config.publicIP,
		Ports:         intToInt32Slice(config.publicPorts),
		Compatibility: compat.Compatibility,
	}
	connConfigCiphertext, err := encryptConnConfigMsg(connConfig, config.privateKey, config.peerPubKey)
	if err != nil {
		return fmt.Errorf("could not encrypt config msg: %v", err)
	}
	endExchangeMsg := &pb.P2PConfigExchangeMsg{
		PublicKey:        config.publicKey.Hex(),
		ConfigCiphertext: connConfigCiphertext,
	}
	log.Debug().Msgf("Consumer %s sending ack with encrypted config to provider %s", consumerID.Address, providerID.Address)
	packedMsg, err := packSignedMsg(m.signer, consumerID, endExchangeMsg)
	if err != nil {
		return fmt.Errorf("could not pack signed message: %v", err)
	}

	// simple broker Publish will not work here since we have to delay Consumer from pinging Provider
	//  until provider receives consumer config ( IP, ports ) and starts pinging Consumer first.
	// This is why we use broker Request method to be sure that Provider processed our given configuration.
	// To improve speed here investigate options to reduce broker communication round trip.
	_, err = m.sendSignedMsg(ctx, configExchangeACKSubject(providerID, serviceType), packedMsg, brokerConn)

	if err != nil {
		return fmt.Errorf("could not send signed msg: %v", err)
	}

	return nil
}

func (m *dialer) prepareLocalPorts(config *p2pConnectConfig) (string, []int, error) {
	trace := config.tracer.StartStage("Consumer P2P exchange (ports)")
	defer config.tracer.EndStage(trace)

	// Finally send consumer encrypted and signed connect config in ack message.
	publicIP, err := m.ipResolver.GetPublicIP()
	if err != nil {
		return "", nil, fmt.Errorf("could not get public IP: %v", err)
	}

	localPorts, err := acquireLocalPorts(m.portPool, len(config.peerPorts))
	if err != nil {
		return publicIP, nil, fmt.Errorf("could not acquire local ports: %v", err)
	}

	return publicIP, localPorts, nil
}

func (m *dialer) dialDirect(ctx context.Context, providerID identity.Identity, config *p2pConnectConfig) (*net.UDPConn, *net.UDPConn, error) {
	trace := config.tracer.StartStage("Consumer P2P dial (upnp)")
	defer config.tracer.EndStage(trace)

	log.Debug().Msg("Skipping provider ping")

	ip := defaultInterfaceAddress()
	conn1, err := net.DialUDP("udp4", &net.UDPAddr{IP: net.ParseIP(ip), Port: config.localPorts[0]}, &net.UDPAddr{IP: net.ParseIP(config.peerIP()), Port: config.peerPorts[0]})
	if err != nil {
		return nil, nil, fmt.Errorf("could not create UDP conn for p2p channel: %w", err)
	}
	conn2, err := net.DialUDP("udp4", &net.UDPAddr{IP: net.ParseIP(ip), Port: config.localPorts[1]}, &net.UDPAddr{IP: net.ParseIP(config.peerIP()), Port: config.peerPorts[1]})
	if err != nil {
		return nil, nil, fmt.Errorf("could not create UDP conn for service: %w", err)
	}

	if err := router.ProtectUDPConn(conn1); err != nil {
		return nil, nil, fmt.Errorf("failed to protect udp connection: %w", err)
	}

	if err := router.ProtectUDPConn(conn2); err != nil {
		return nil, nil, fmt.Errorf("failed to protect udp connection: %w", err)
	}

	return conn1, conn2, err
}

func (m *dialer) dialPinger(ctx context.Context, providerID identity.Identity, config *p2pConnectConfig) (*net.UDPConn, *net.UDPConn, error) {
	trace := config.tracer.StartStage("Consumer P2P dial (pinger)")
	defer config.tracer.EndStage(trace)

	if _, err := firewall.AllowIPAccess(config.peerPublicIP); err != nil {
		return nil, nil, fmt.Errorf("could not add peer IP firewall rule: %w", err)
	}

	ip := defaultInterfaceAddress()
	log.Debug().Msgf("Pinging provider %s  using ports %v:%v", providerID.Address, config.localPorts, config.peerPorts)
	conns, err := m.consumerPinger.PingProviderPeer(ctx, ip, config.peerIP(), config.localPorts, config.peerPorts, consumerInitialTTL, requiredConnCount)
	if err != nil {
		return nil, nil, fmt.Errorf("could not ping peer: %w", err)
	}
	return conns[0], conns[1], nil
}

func (m *dialer) sendSignedMsg(ctx context.Context, subject string, msg []byte, brokerConn nats.Connection) ([]byte, error) {
	reply, err := brokerConn.RequestWithContext(ctx, subject, msg)
	if err != nil {
		return nil, fmt.Errorf("could not send broker request to subject %s: %v", subject, err)
	}
	return reply.Data, nil
}

func (m *dialer) channelHandlersReady(msg *nats_lib.Msg) error {
	var handlersReady pb.P2PChannelHandlersReady
	if err := proto.Unmarshal(msg.Data, &handlersReady); err != nil {
		return fmt.Errorf("failed to unmarshal handlers ready message: %w", err)
	}
	if handlersReady.Value != "HANDLERS READY" {
		return errors.New("incorrect handlers ready message value")
	}

	return nil
}
