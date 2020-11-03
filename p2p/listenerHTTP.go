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
	"io/ioutil"
	"net"
	"net/http"
	"sync"

	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat/mapping"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/trace"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

// NewListener creates new p2p communication listener which is used on provider side.
func NewListenerHTTP(address string, signer identity.SignerFactory, verifier identity.Verifier, ipResolver ip.Resolver, providerPinger natProviderPinger, portPool port.ServicePortSupplier, portMapper mapping.PortMapper) Listener {
	return &listenerHTTP{
		address:        address,
		pendingConfigs: map[PublicKey]p2pConnectConfig{},
		ipResolver:     ipResolver,
		signer:         signer,
		verifier:       verifier,
		portPool:       portPool,
		providerPinger: providerPinger,
		portMapper:     portMapper,
	}
}

// listenerHTTP implements Listener interface.
type listenerHTTP struct {
	address        string
	portPool       port.ServicePortSupplier
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

func (m *listenerHTTP) GetContact() market.Contact {
	return market.Contact{
		Type:       ContactTypeHTTPV1,
		Definition: ContactDefinition{BrokerAddresses: []string{m.address}}}
}

type events struct {
	Events []struct {
		Timestamp int64  `json:"timestamp"`
		Category  string `json:"category"`
		Data      []byte `json:"data"`
	} `json:"events"`
}

// Listen listens for incoming peer connections to establish new p2p channels. Establishes p2p channel and passes it
// to channelHandlers.
func (m *listenerHTTP) Listen(providerID identity.Identity, serviceType string, channelHandlers func(ch Channel)) (func(), error) {
	outboundIP, err := m.ipResolver.GetOutboundIP()
	if err != nil {
		return func() {}, fmt.Errorf("could not get outbound IP: %w", err)
	}

	go func() {
		for {
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/poll/init/%s", m.address, providerID.Address), nil)
			if err != nil {
				panic(err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				panic(err)
			}

			var events events
			if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
				panic(err)
			}

			if len(events.Events) == 0 {
				continue
			}

			msg := events.Events[0].Data
			if err := m.providerStartConfigExchange(providerID, msg, outboundIP); err != nil {
				log.Err(err).Msg("Could not handle initial exchange")
				return
			}
		}
	}()

	go func() {
		for {
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/poll/ack/%s", m.address, providerID.Address), nil)
			if err != nil {
				panic(err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				panic(err)
			}

			var events events
			if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
				if err != io.EOF {
					panic(err)
				}
				continue
			}

			if len(events.Events) == 0 {
				continue
			}
			msg := events.Events[0].Data

			config, err := m.providerAckConfigExchange(msg)
			if err != nil {
				log.Err(err).Msg("Could not handle exchange ack")
				return
			}

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
				conns, err := m.providerPinger.PingConsumerPeer(context.Background(), config.peerIP(), config.localPorts, config.peerPorts, providerInitialTTL, requiredConnCount)
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

			config.tracer.EndStage(traceAck)
		}
	}()

	return func() {}, nil
}

func (m *listenerHTTP) providerStartConfigExchange(providerID identity.Identity, msg []byte, outboundIP string) error {
	tracer := trace.NewTracer("Provider whole Connect")

	trace := tracer.StartStage("Provider P2P exchange")
	defer tracer.EndStage(trace)

	pubKey, privateKey, err := GenerateKey()
	if err != nil {
		return fmt.Errorf("could not generate provider p2p keys: %w", err)
	}

	// Get initial peer exchange with it's public key.
	signedMsg, err := unpackSignedMsg(m.verifier, msg)
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

	publicIP, localPorts, portsRelease, err := m.prepareLocalPorts(outboundIP, tracer)
	if err != nil {
		return fmt.Errorf("could not prepare ports: %w", err)
	}

	m.setPendingConfig(p2pConnectConfig{
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
	packedMsg, err := packSignedMsg(m.signer, providerID, &exchangeMsg)
	if err != nil {
		return fmt.Errorf("could not pack signed message: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/msg/%s?type=push", m.address, providerID.Address), bytes.NewBuffer(packedMsg))
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not send http broker request: %w", err)
	}

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return nil
}

// prepareLocalPorts acquires ports for p2p connections. It tries to acquire only
// required ports count for actual p2p and service connections and fallback to
// acquiring extra ports for nat pinger if provider is behind nat, port mapping failed
// and no manual port forwarding is enabled.
func (m *listenerHTTP) prepareLocalPorts(outboundIP string, tracer *trace.Tracer) (string, []int, []func(), error) {
	trace := tracer.StartStage("Provider P2P exchange (ports)")
	defer tracer.EndStage(trace)

	// Send reply with encrypted exchange config.
	publicIP, err := m.ipResolver.GetPublicIP()
	if err != nil {
		return "", nil, nil, fmt.Errorf("could not get public IP: %v", err)
	}

	// First acquire required only ports for needed n connections.
	localPorts, err := acquireLocalPorts(m.portPool, requiredConnCount)
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
		portRelease, portMappingOk = m.portMapper.Map("UDP", p, "Myst node p2p port mapping")
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
	if _, noop := m.providerPinger.(*traversal.NoopPinger); noop {
		return publicIP, localPorts, nil, nil
	}

	// Acquire more ports for nat pinger.
	morePorts, err := acquireLocalPorts(m.portPool, pingMaxPorts-requiredConnCount)
	if err != nil {
		return publicIP, nil, nil, fmt.Errorf("could not acquire more local ports: %v", err)
	}
	for _, p := range morePorts {
		localPorts = append(localPorts, p)
	}
	return publicIP, localPorts, nil, nil
}

func (m *listenerHTTP) providerAckConfigExchange(msg []byte) (*p2pConnectConfig, error) {
	signedMsg, err := unpackSignedMsg(m.verifier, msg)
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

func (m *listenerHTTP) pendingConfig(peerPubKey PublicKey) (p2pConnectConfig, bool) {
	m.pendingConfigsMu.Lock()
	defer m.pendingConfigsMu.Unlock()
	config, ok := m.pendingConfigs[peerPubKey]
	return config, ok
}

func (m *listenerHTTP) setPendingConfig(config p2pConnectConfig) {
	m.pendingConfigsMu.Lock()
	defer m.pendingConfigsMu.Unlock()
	m.pendingConfigs[config.peerPubKey] = config
}

func (m *listenerHTTP) deletePendingConfig(peerPubKey PublicKey) {
	m.pendingConfigsMu.Lock()
	defer m.pendingConfigsMu.Unlock()
	delete(m.pendingConfigs, peerPubKey)
}
