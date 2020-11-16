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

	"github.com/mysteriumnetwork/node/trace"
	nats_lib "github.com/nats-io/nats.go"

	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/pb"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

const maxBrokerConnectAttempts = 25

// NewDialerNATS creates new p2p communication dialer which is used on consumer side.
func NewDialerNATS(broker brokerConnector, signer identity.SignerFactory, verifier identity.Verifier, ipResolver ip.Resolver, consumerPinger natConsumerPinger, portPool port.ServicePortSupplier) *dialerNATS {
	return &dialerNATS{
		broker:         broker,
		ipResolver:     ipResolver,
		signer:         signer,
		verifier:       verifier,
		portPool:       portPool,
		consumerPinger: consumerPinger,
	}
}

// dialerNATS implements Dialer interface.
type dialerNATS struct {
	portPool       port.ServicePortSupplier
	broker         brokerConnector
	consumerPinger natConsumerPinger
	signer         identity.SignerFactory
	verifier       identity.Verifier
	ipResolver     ip.Resolver
}

// Dial exchanges p2p configuration via broker, performs NAT pinging if needed
// and create p2p channel which is ready for communication.
func (d *dialerNATS) Dial(ctx context.Context, consumerID, providerID identity.Identity, serviceType string, contactDef ContactDefinition, tracer *trace.Tracer) (Channel, error) {
	config := &p2pConnectConfig{tracer: tracer}

	// Send initial exchange with signed consumer public key.
	brokerConn, err := d.connect(contactDef, tracer)
	if err != nil {
		return nil, fmt.Errorf("could not open broker conn: %w", err)
	}
	defer brokerConn.Close()

	peerReady := make(chan struct{})
	var once sync.Once
	_, err = brokerConn.Subscribe(channelHandlersReadySubject(providerID, serviceType), func(msg *nats_lib.Msg) {
		defer once.Do(func() { close(peerReady) })
		if err := d.channelHandlersReady(msg); err != nil {
			log.Err(err).Msg("Channel handlers ready handler setup failed")
			return
		}
	})

	config, err = d.startConfigExchange(config, ctx, brokerConn, providerID, serviceType, consumerID)
	if err != nil {
		return nil, fmt.Errorf("could not exchange config: %w", err)
	}

	config.publicIP, config.localPorts, err = d.prepareLocalPorts(config)
	if err != nil {
		return nil, fmt.Errorf("could not prepare ports: %w", err)
	}

	// Finally send consumer encrypted and signed connect config in ack message.
	err = d.ackConfigExchange(config, ctx, brokerConn, providerID, serviceType, consumerID)
	if err != nil {
		return nil, fmt.Errorf("could not ack config: %w", err)
	}

	dial := d.dialPinger
	if len(config.peerPorts) == requiredConnCount {
		dial = dialDirect
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

	channel, err := newChannel(conn1, config.privateKey, config.peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("could not create p2p channel during dial: %w", err)
	}
	channel.setTracer(tracer)
	channel.setServiceConn(conn2)
	channel.launchReadSendLoops()
	config.tracer.EndStage(traceAck)

	return channel, nil
}

func (d *dialerNATS) connect(contactDef ContactDefinition, tracer *trace.Tracer) (conn nats.Connection, err error) {
	trace := tracer.StartStage("Consumer P2P connect")
	defer tracer.EndStage(trace)

	// broker connect might fail due to reconfiguration of network routes in progress
	for i := 0; i < maxBrokerConnectAttempts; i++ {
		serverURLs, err := nats.ParseServerURIs(contactDef.BrokerAddresses)
		if err != nil {
			return nil, err
		}

		conn, err = d.broker.Connect(serverURLs...)
		if err != nil {
			log.Warn().Msgf("broker connect failed - attempting again in 1sec: %s", err)
			time.Sleep(time.Second)
			continue
		}
		break
	}
	return conn, err
}

func (d *dialerNATS) startConfigExchange(config *p2pConnectConfig, ctx context.Context, brokerConn nats.Connection, providerID identity.Identity, serviceType string, consumerID identity.Identity) (*p2pConnectConfig, error) {
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
	packedMsg, err := packSignedMsg(d.signer, consumerID, beginExchangeMsg)
	if err != nil {
		return nil, fmt.Errorf("could not pack signed message: %v", err)
	}
	exchangeMsgBrokerReply, err := d.sendSignedMsg(ctx, configExchangeSubject(providerID, serviceType), packedMsg, brokerConn)
	if err != nil {
		return nil, fmt.Errorf("could not send signed message: %w", err)
	}

	// Parse provider response with public key and encrypted and signed connection config.
	exchangeMsgReplySignedMsg, err := unpackSignedMsg(d.verifier, exchangeMsgBrokerReply)
	if err != nil {
		return nil, fmt.Errorf("could not unpack peer siged message: %w", err)
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
	log.Debug().Msgf("Consumer %s received provider %s with config: %v", consumerID.Address, providerID.Address, peerConnConfig)

	config.publicKey = pubKey
	config.privateKey = privateKey
	config.peerPubKey = peerPubKey
	config.peerPublicIP = peerConnConfig.PublicIP
	config.peerPorts = int32ToIntSlice(peerConnConfig.Ports)
	return config, nil
}

func (d *dialerNATS) ackConfigExchange(config *p2pConnectConfig, ctx context.Context, brokerConn nats.Connection, providerID identity.Identity, serviceType string, consumerID identity.Identity) error {
	trace := config.tracer.StartStage("Consumer P2P exchange ack")
	defer config.tracer.EndStage(trace)

	connConfig := &pb.P2PConnectConfig{
		PublicIP: config.publicIP,
		Ports:    intToInt32Slice(config.localPorts),
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
	packedMsg, err := packSignedMsg(d.signer, consumerID, endExchangeMsg)
	if err != nil {
		return fmt.Errorf("could not pack signed message: %v", err)
	}

	// simple broker Publish will not work here since we have to delay Consumer from pinging Provider
	//  until provider receives consumer config ( IP, ports ) and starts pinging Consumer first.
	// This is why we use broker Request method to be sure that Provider processed our given configuration.
	// To improve speed here investigate options to reduce broker communication round trip.
	_, err = d.sendSignedMsg(ctx, configExchangeACKSubject(providerID, serviceType), packedMsg, brokerConn)

	if err != nil {
		return fmt.Errorf("could not send signed msg: %v", err)
	}

	return nil
}

func (d *dialerNATS) prepareLocalPorts(config *p2pConnectConfig) (string, []int, error) {
	trace := config.tracer.StartStage("Consumer P2P exchange (ports)")
	defer config.tracer.EndStage(trace)

	// Finally send consumer encrypted and signed connect config in ack message.
	publicIP, err := d.ipResolver.GetPublicIP()
	if err != nil {
		return "", nil, fmt.Errorf("could not get public IP: %v", err)
	}

	localPorts, err := acquireLocalPorts(d.portPool, len(config.peerPorts))
	if err != nil {
		return publicIP, nil, fmt.Errorf("could not acquire local ports: %v", err)
	}

	return publicIP, localPorts, nil
}

func dialDirect(ctx context.Context, providerID identity.Identity, config *p2pConnectConfig) (*net.UDPConn, *net.UDPConn, error) {
	trace := config.tracer.StartStage("Consumer P2P dial (upnp)")
	defer config.tracer.EndStage(trace)

	if _, err := firewall.AllowIPAccess(config.peerPublicIP); err != nil {
		return nil, nil, fmt.Errorf("could not add peer IP firewall rule: %w", err)
	}

	log.Debug().Msg("Skipping provider ping")
	conn1, err := net.DialUDP("udp4", &net.UDPAddr{Port: config.localPorts[0]}, &net.UDPAddr{IP: net.ParseIP(config.peerIP()), Port: config.peerPorts[0]})
	if err != nil {
		return nil, nil, fmt.Errorf("could not create UDP conn for p2p channel: %w", err)
	}
	conn2, err := net.DialUDP("udp4", &net.UDPAddr{Port: config.localPorts[1]}, &net.UDPAddr{IP: net.ParseIP(config.peerIP()), Port: config.peerPorts[1]})
	if err != nil {
		return nil, nil, fmt.Errorf("could not create UDP conn for service: %w", err)
	}
	return conn1, conn2, err
}

func (d *dialerNATS) dialPinger(ctx context.Context, providerID identity.Identity, config *p2pConnectConfig) (*net.UDPConn, *net.UDPConn, error) {
	trace := config.tracer.StartStage("Consumer P2P dial (pinger)")
	defer config.tracer.EndStage(trace)

	if _, err := firewall.AllowIPAccess(config.peerPublicIP); err != nil {
		return nil, nil, fmt.Errorf("could not add peer IP firewall rule: %w", err)
	}

	log.Debug().Msgf("Pinging provider %s with IP %s using ports %v:%v", providerID.Address, config.peerIP(), config.localPorts, config.peerPorts)
	conns, err := d.consumerPinger.PingProviderPeer(ctx, config.peerIP(), config.localPorts, config.peerPorts, consumerInitialTTL, requiredConnCount)
	if err != nil {
		return nil, nil, fmt.Errorf("could not ping peer: %w", err)
	}
	return conns[0], conns[1], nil
}

func (d *dialerNATS) sendSignedMsg(ctx context.Context, subject string, msg []byte, brokerConn nats.Connection) ([]byte, error) {
	reply, err := brokerConn.RequestWithContext(ctx, subject, msg)
	if err != nil {
		return nil, fmt.Errorf("could not send broker request to subject %s: %v", subject, err)
	}
	return reply.Data, nil
}

func (d *dialerNATS) channelHandlersReady(msg *nats_lib.Msg) error {
	var handlersReady pb.P2PChannelHandlersReady
	if err := proto.Unmarshal(msg.Data, &handlersReady); err != nil {
		return fmt.Errorf("failed to unmarshal handlers ready message: %w", err)
	}
	if handlersReady.Value != "HANDLERS READY" {
		return errors.New("incorrect handlers ready message value")
	}

	return nil
}
