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

	nats_lib "github.com/nats-io/go-nats"

	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/nat/traversal"

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
	Dial(ctx context.Context, consumerID, providerID identity.Identity, serviceType string, contactDef ContactDefinition) (Channel, error)
}

// NewDialer creates new p2p communication dialer which is used on consumer side.
func NewDialer(broker brokerConnector, signer identity.SignerFactory, verifier identity.Verifier, ipResolver ip.Resolver, consumerPinger natConsumerPinger, portPool port.ServicePortSupplier) Dialer {
	return &dialer{
		broker:         broker,
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
}

// Dial exchanges p2p configuration via broker, performs NAT pinging if needed
// and create p2p channel which is ready for communication.
func (m *dialer) Dial(ctx context.Context, consumerID, providerID identity.Identity, serviceType string, contactDef ContactDefinition) (Channel, error) {
	brokerConn, err := m.broker.Connect(contactDef.BrokerAddresses...)
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

	config, err := m.exchangeConfig(ctx, brokerConn, providerID, serviceType, consumerID)
	if err != nil {
		return nil, fmt.Errorf("could not exchange config: %w", err)
	}

	if _, err := firewall.AllowIPAccess(config.peerPublicIP); err != nil {
		return nil, fmt.Errorf("could not add peer IP firewall rule: %w", err)
	}

	var conn1, conn2 *net.UDPConn
	if len(config.peerPorts) == requiredConnCount {
		log.Debug().Msg("Skipping provider ping")
		conn1, err = net.DialUDP("udp4", &net.UDPAddr{Port: config.localPorts[0]}, &net.UDPAddr{IP: net.ParseIP(config.peerIP()), Port: config.peerPorts[0]})
		if err != nil {
			return nil, fmt.Errorf("could not create UDP conn for p2p channel: %w", err)
		}
		conn2, err = net.DialUDP("udp4", &net.UDPAddr{Port: config.localPorts[1]}, &net.UDPAddr{IP: net.ParseIP(config.peerIP()), Port: config.peerPorts[1]})
		if err != nil {
			return nil, fmt.Errorf("could not create UDP conn for service: %w", err)
		}
	} else {
		// race condition still happens when consumer starts to ping until provider did not manage to complete required number of pings
		// this might be provider / consumer performance dependent
		// make sleep time dependent on pinger interval and wait for 2 ping iterations
		// TODO: either reintroduce eventual increase of TTL on consumer or maintain some sane delay
		dur := traversal.DefaultPingConfig().Interval.Milliseconds() * 2
		log.Debug().Msgf("Sleeping for %v ms - waiting for provider to launch pings", dur)
		time.Sleep(time.Duration(dur) * time.Millisecond)
		log.Debug().Msgf("Pinging provider %s with IP %s using ports %v:%v", providerID.Address, config.peerIP(), config.localPorts, config.peerPorts)
		conns, err := m.consumerPinger.PingProviderPeer(ctx, config.peerIP(), config.localPorts, config.peerPorts, consumerInitialTTL, requiredConnCount)
		if err != nil {
			return nil, fmt.Errorf("could not ping peer: %w", err)
		}
		conn1 = conns[0]
		conn2 = conns[1]
	}

	// Wait until provider confirms that channel handlers are ready.
	select {
	case <-peerReady:
		log.Debug().Msg("Received handlers ready message from provider")
	case <-ctx.Done():
		return nil, errors.New("timeout while performing configuration exchange")
	}

	channel, err := newChannel(conn1, config.privateKey, config.peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("could not create p2p channel: %w", err)
	}
	channel.setServiceConn(conn2)
	return channel, nil
}

func (m *dialer) exchangeConfig(ctx context.Context, brokerConn nats.Connection, providerID identity.Identity, serviceType string, consumerID identity.Identity) (*p2pConnectConfig, error) {
	pubKey, privateKey, err := GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("could not generate consumer p2p keys: %w", err)
	}

	// Send initial exchange with signed consumer public key.
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
	localPorts, err := acquireLocalPorts(m.portPool, len(peerConnConfig.Ports))
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
	_, err = m.sendSignedMsg(ctx, configExchangeACKSubject(providerID, serviceType), packedMsg, brokerConn)
	if err != nil {
		return nil, fmt.Errorf("could not send signed msg: %v", err)
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

func (m *dialer) sendSignedMsg(ctx context.Context, subject string, msg []byte, brokerConn nats.Connection) ([]byte, error) {
	reply, err := brokerConn.RequestWithContext(ctx, subject, msg)
	if err != nil {
		return nil, fmt.Errorf("could send broker request to subject %s: %v", subject, err)
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
