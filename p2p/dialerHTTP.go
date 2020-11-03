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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/mysteriumnetwork/node/trace"

	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/pb"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

const (
	MsgTypeInit = "init"
	MsgTypeAck  = "ack"
)

// NewDialer creates new p2p communication dialer which is used on consumer side.
func NewDialerHTTP(signer identity.SignerFactory, verifier identity.Verifier, ipResolver ip.Resolver, consumerPinger natConsumerPinger, portPool port.ServicePortSupplier) Dialer {
	return &dialerHTTP{
		ipResolver:     ipResolver,
		signer:         signer,
		verifier:       verifier,
		portPool:       portPool,
		consumerPinger: consumerPinger,
	}
}

// dialer implements Dialer interface.
type dialerHTTP struct {
	portPool       port.ServicePortSupplier
	consumerPinger natConsumerPinger
	signer         identity.SignerFactory
	verifier       identity.Verifier
	ipResolver     ip.Resolver
}

// Dial exchanges p2p configuration via broker, performs NAT pinging if needed
// and create p2p channel which is ready for communication.
func (m *dialerHTTP) Dial(ctx context.Context, consumerID, providerID identity.Identity, serviceType string, contactDef ContactDefinition, tracer *trace.Tracer) (Channel, error) {
	config := &p2pConnectConfig{tracer: tracer}
	config, err := m.startConfigExchange(config, ctx, contactDef, providerID, serviceType, consumerID)
	if err != nil {
		return nil, fmt.Errorf("could not exchange config: %w", err)
	}

	config.publicIP, config.localPorts, err = m.prepareLocalPorts(config)
	if err != nil {
		return nil, fmt.Errorf("could not prepare ports: %w", err)
	}

	// Finally send consumer encrypted and signed connect config in ack message.
	err = m.ackConfigExchange(config, ctx, contactDef, providerID, serviceType, consumerID)
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

func (m *dialerHTTP) startConfigExchange(config *p2pConnectConfig, ctx context.Context, contactDef ContactDefinition, providerID identity.Identity, serviceType string, consumerID identity.Identity) (*p2pConnectConfig, error) {
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
	url := fmt.Sprintf("%s/msg/%s?type=%s", contactDef.BrokerAddresses[0], providerID.Address, MsgTypeInit)
	exchangeMsgBrokerReply, err := m.sendSignedMsg(ctx, url, packedMsg)
	if err != nil {
		return nil, fmt.Errorf("could not send signed message: %w", err)
	}

	// Parse provider response with public key and encrypted and signed connection config.
	exchangeMsgReplySignedMsg, err := unpackSignedMsg(m.verifier, exchangeMsgBrokerReply)
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
	log.Debug().Msgf("Consumer %s received provider %s with config: %v", consumerID.Address, providerID.Address, peerConnConfig)

	config.publicKey = pubKey
	config.privateKey = privateKey
	config.peerPubKey = peerPubKey
	config.peerPublicIP = peerConnConfig.PublicIP
	config.peerPorts = int32ToIntSlice(peerConnConfig.Ports)
	return config, nil
}

func (m *dialerHTTP) ackConfigExchange(config *p2pConnectConfig, ctx context.Context, contactDef ContactDefinition, providerID identity.Identity, serviceType string, consumerID identity.Identity) error {
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
	packedMsg, err := packSignedMsg(m.signer, consumerID, endExchangeMsg)
	if err != nil {
		return fmt.Errorf("could not pack signed message: %v", err)
	}

	url := fmt.Sprintf("%s/msg/%s?type=%s", contactDef.BrokerAddresses[0], providerID.Address, MsgTypeAck)
	_, err = m.sendSignedMsg(ctx, url, packedMsg)

	if err != nil {
		return fmt.Errorf("could not send signed msg: %v", err)
	}

	return nil
}

func (m *dialerHTTP) prepareLocalPorts(config *p2pConnectConfig) (string, []int, error) {
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

func (m *dialerHTTP) dialDirect(ctx context.Context, providerID identity.Identity, config *p2pConnectConfig) (*net.UDPConn, *net.UDPConn, error) {
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

func (m *dialerHTTP) dialPinger(ctx context.Context, providerID identity.Identity, config *p2pConnectConfig) (*net.UDPConn, *net.UDPConn, error) {
	trace := config.tracer.StartStage("Consumer P2P dial (pinger)")
	defer config.tracer.EndStage(trace)

	if _, err := firewall.AllowIPAccess(config.peerPublicIP); err != nil {
		return nil, nil, fmt.Errorf("could not add peer IP firewall rule: %w", err)
	}

	log.Debug().Msgf("Pinging provider %s with IP %s using ports %v:%v", providerID.Address, config.peerIP(), config.localPorts, config.peerPorts)
	conns, err := m.consumerPinger.PingProviderPeer(ctx, config.peerIP(), config.localPorts, config.peerPorts, consumerInitialTTL, requiredConnCount)
	if err != nil {
		return nil, nil, fmt.Errorf("could not ping peer: %w", err)
	}
	return conns[0], conns[1], nil
}

func (m *dialerHTTP) sendSignedMsg(ctx context.Context, url string, msg []byte) ([]byte, error) {
	log.Info().Err(nil).Msgf("Consumer send %s len %d", url, len(msg))
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(msg))
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not send broker HTTP request: %w", err)
	}

	var events events
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		if err != io.EOF {
			panic(err)
		}

		return []byte{}, nil
	}
	if err != nil {
		return nil, err
	}

	res := events.Events[0].Data
	return res, nil
}
