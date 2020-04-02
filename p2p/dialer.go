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
	"errors"
	"fmt"
	"net"
	"time"

	nats_lib "github.com/nats-io/go-nats"

	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/pb"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

// Dialer knows how to exchange p2p keys and encrypted configuration and creates ready to use p2p channels.
type Dialer interface {
	// Dial exchanges p2p configuration via broker, performs NAT pinging if needed
	// and create p2p channel which is ready for communication.
	Dial(consumerID, providerID identity.Identity, serviceType string, timeout time.Duration) (Channel, error)
}

// NewDialer creates new p2p communication dialer which is used on consumer side.
func NewDialer(broker brokerConnector, address string, signer identity.SignerFactory, verifier identity.Verifier, ipResolver ip.Resolver, consumerPinger natConsumerPinger, portPool port.ServicePortSupplier) Dialer {
	return &dialer{
		broker:         broker,
		brokerAddress:  address,
		ipResolver:     ipResolver,
		signer:         signer,
		verifier:       verifier,
		portPool:       portPool,
		consumerPinger: consumerPinger,
	}
}

// dialer implements Dialer interface.
type dialer struct {
	portPool       port.ServicePortSupplier
	broker         brokerConnector
	consumerPinger natConsumerPinger
	signer         identity.SignerFactory
	verifier       identity.Verifier
	ipResolver     ip.Resolver
	brokerAddress  string
}

// Dial exchanges p2p configuration via broker, performs NAT pinging if needed
// and create p2p channel which is ready for communication.
func (m *dialer) Dial(consumerID, providerID identity.Identity, serviceType string, timeout time.Duration) (Channel, error) {
	config, err := m.exchangeConfig(consumerID, providerID, serviceType, timeout)
	if err != nil {
		return nil, fmt.Errorf("could not exchange config: %w", err)
	}

	var remotePort, localPort int
	var conn0 *net.UDPConn
	var serviceConn *net.UDPConn
	if len(config.peerPorts) == 1 {
		localPort = config.localPorts[0]
		remotePort = config.peerPorts[0]
	} else {
		log.Debug().Msgf("Pinging provider %s with IP %s using ports %v:%v", providerID.Address, config.pingIP(), config.localPorts, config.peerPorts)
		conns, err := m.consumerPinger.PingProviderPeer(config.pingIP(), config.localPorts, config.peerPorts, consumerInitialTTL, requiredConnAmount)
		if err != nil {
			return nil, fmt.Errorf("could not ping peer: %w", err)
		}
		conn0 = conns[0]
		localPort = conn0.LocalAddr().(*net.UDPAddr).Port
		remotePort = conn0.RemoteAddr().(*net.UDPAddr).Port
		serviceConn = conns[1]
		log.Debug().Msgf("Will use service conn with local port: %d, remote port: %d", serviceConn.LocalAddr().(*net.UDPAddr).Port, serviceConn.RemoteAddr().(*net.UDPAddr).Port)
	}

	log.Debug().Msgf("Creating channel with listen port: %d, peer port: %d", localPort, remotePort)
	channel, err := newChannel(conn0, config.privateKey, config.peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("could not create p2p channel: %w", err)
	}
	channel.serviceConn = serviceConn
	return channel, nil
}

func (m *dialer) exchangeConfig(consumerID, providerID identity.Identity, serviceType string, timeout time.Duration) (*p2pConnectConfig, error) {
	pubKey, privateKey, err := GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("could not generate consumer p2p keys: %w", err)
	}

	brokerConn, err := m.broker.Connect(m.brokerAddress)
	if err != nil {
		return nil, fmt.Errorf("could not open broker conn: %w", err)
	}
	defer brokerConn.Close()

	ready := make(chan bool)
	_, err = brokerConn.Subscribe(channelHandlersReadySubject(providerID, serviceType), func(msg *nats_lib.Msg) {
		defer close(ready)
		if err := m.channelHandlersReady(msg); err != nil {
			log.Err(err).Msg("Channel handlers ready handler setup failed")
			return
		}
	})

	// Send initial exchange with signed consumer public key.
	beginExchangeMsg := &pb.P2PConfigExchangeMsg{
		PublicKey: pubKey.Hex(),
	}
	log.Debug().Msgf("Consumer %s sending public key %s to provider %s", consumerID.Address, beginExchangeMsg.PublicKey, providerID.Address)
	packedMsg, err := packSignedMsg(m.signer, consumerID, beginExchangeMsg)
	if err != nil {
		return nil, fmt.Errorf("could not pack signed message: %v", err)
	}
	exchangeMsgBrokerReply, err := m.sendSignedMsg(brokerConn, configExchangeSubject(providerID, serviceType), packedMsg, timeout)
	if err != nil {
		return nil, fmt.Errorf("could not send signed message: %w", err)
	}

	// Parse provider response with public key and encrypted and signed connection config.
	exchangeMsgReplySignedMsg, err := unpackSignedMsg(m.verifier, exchangeMsgBrokerReply)
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

	// Finally send consumer encrypted and signed connect config in ack message.
	publicIP, err := m.ipResolver.GetPublicIP()
	if err != nil {
		return nil, fmt.Errorf("could not get public IP: %v", err)
	}
	localPorts, err := acquireLocalPorts(m.portPool)
	if err != nil {
		return nil, fmt.Errorf("could not acquire local ports: %v", err)
	}
	connConfig := &pb.P2PConnectConfig{
		PublicIP: publicIP,
		Ports:    intToInt32Slice(localPorts),
	}
	connConfigCiphertext, err := encryptConnConfigMsg(connConfig, privateKey, peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("could not encrypt config msg: %v", err)
	}
	endExchangeMsg := &pb.P2PConfigExchangeMsg{
		PublicKey:        pubKey.Hex(),
		ConfigCiphertext: connConfigCiphertext,
	}
	log.Debug().Msgf("Consumer %s sending ack with encrypted config to provider %s", consumerID.Address, providerID.Address)
	packedMsg, err = packSignedMsg(m.signer, consumerID, endExchangeMsg)
	if err != nil {
		return nil, fmt.Errorf("could not pack signed message: %v", err)
	}
	_, err = m.sendSignedMsg(brokerConn, configExchangeACKSubject(providerID, serviceType), packedMsg, timeout)
	if err != nil {
		return nil, fmt.Errorf("could not send signed msg: %v", err)
	}

	// TODO: put this under feature set check when its available
	// wait until provider confirms that channel handlers are ready
	select {
	case <- ready:
		log.Debug().Msg("Received handlers ready message from provider")
	// NOTE: We can make this timeout much larger when all providers migrate to handlers ready message
	case <- time.After(1*time.Second):
		log.Debug().Msg("Failed to receive handlers ready message from provider - continuing")
	}

	return &p2pConnectConfig{
		publicIP:     publicIP,
		privateKey:   privateKey,
		localPorts:   localPorts,
		peerPubKey:   peerPubKey,
		peerPublicIP: peerConnConfig.PublicIP,
		peerPorts:    int32ToIntSlice(peerConnConfig.Ports),
	}, nil
}

func (m *dialer) sendSignedMsg(brokerConn nats.Connection, subject string, msg []byte, timeout time.Duration) ([]byte, error) {
	reply, err := brokerConn.Request(subject, msg, timeout)
	if err != nil {
		return nil, fmt.Errorf("could send broker request to subject %s: %v", subject, err)
	}
	return reply.Data, nil
}

func (m *dialer) channelHandlersReady(msg *nats_lib.Msg) error {
	var handlersReady pb.P2PChannelHandlersReady
	if err := proto.Unmarshal(msg.Data, &handlersReady); err != nil {
		return fmt.Errorf("failed to unmarshal handlers ready message %w", err)
	}
	if handlersReady.Value != "HANDLERS READY" {
		return errors.New("incorrect handlers ready message value")
	}

	return nil
}
