//+build !windows

/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package service

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/dns"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/nat/event"
	"github.com/mysteriumnetwork/node/nat/mapping"
	"github.com/mysteriumnetwork/node/nat/traversal"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/utils/netutil"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// NATPinger defined Pinger interface for Provider
type NATPinger interface {
	BindServicePort(key string, port int)
	Stop()
	Valid() bool
}

// NATEventGetter allows us to fetch the last known NAT event
type NATEventGetter interface {
	LastEvent() *event.Event
}

// NewManager creates new instance of Wireguard service
func NewManager(
	ipResolver ip.Resolver,
	location location.ServiceLocationInfo,
	natService nat.NATService,
	natPinger NATPinger,
	natEventGetter NATEventGetter,
	portMap func(port int) (releasePortMapping func()),
	options Options,
	portSupplier port.ServicePortSupplier,
) *Manager {
	resourceAllocator := resources.NewAllocator(portSupplier, options.Subnet)
	return &Manager{
		done:           make(chan struct{}),
		ipResolver:     ipResolver,
		natService:     natService,
		natPinger:      natPinger,
		natEventGetter: natEventGetter,
		natPingerPorts: port.NewPool(),
		connectionEndpointFactory: func() (wg.ConnectionEndpoint, error) {
			return endpoint.NewConnectionEndpoint(ipResolver, resourceAllocator, portMap, options.ConnectDelay)
		},
		publicIP:   location.PubIP,
		outboundIP: location.OutIP,
	}
}

// Manager represents an instance of Wireguard service
type Manager struct {
	done chan struct{}

	natService     nat.NATService
	natPinger      NATPinger
	natPingerPorts port.ServicePortSupplier
	natEventGetter NATEventGetter

	dnsOK      bool
	dnsPort    int
	dnsProxy   *dns.Proxy
	dnsProxyMu sync.Mutex

	connectionEndpointFactory func() (wg.ConnectionEndpoint, error)

	ipResolver ip.Resolver
	publicIP   string
	outboundIP string
}

// ProvideConfig provides the config for consumer and handles new WireGuard connection.
func (m *Manager) ProvideConfig(sessionConfig json.RawMessage) (*session.ConfigParams, error) {
	consumerConfig := wg.ConsumerConfig{}
	err := json.Unmarshal(sessionConfig, &consumerConfig)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal wg consumer config")
	}

	natPingerEnabled := m.natPinger.Valid() && m.isBehindNAT() && m.portMappingFailed()
	traversalParams, err := m.newTraversalParams(natPingerEnabled, consumerConfig)
	if err != nil {
		return nil, errors.Wrap(err, "could not create traversal params")
	}

	conn, err := m.startNewConnection()
	if err != nil {
		return nil, errors.Wrap(err, "could not start new connection")
	}

	config, err := conn.Config()
	if err != nil {
		return nil, errors.Wrap(err, "could not get peer config")
	}

	if natPingerEnabled {
		log.Info().Msgf("NAT Pinger enabled, binding service port for proxy, key: %s", traversalParams.ProxyPortMappingKey)
		m.natPinger.BindServicePort(traversalParams.ProxyPortMappingKey, config.Provider.Endpoint.Port)
		newConfig, err := m.addTraversalParams(config, traversalParams)
		if err != nil {
			return nil, errors.Wrap(err, "could not apply NAT traversal params")
		}
		config = newConfig
	}

	if err := m.addConsumerPeer(conn, traversalParams.ConsumerPort, traversalParams.ProviderPort, consumerConfig.PublicKey); err != nil {
		return nil, errors.Wrap(err, "could not add consumer peer")
	}

	var dnsIP net.IP
	if m.dnsOK {
		dnsIP = netutil.FirstIP(config.Consumer.IPAddress)
		config.Consumer.DNSIPs = dnsIP.String()
	}

	natRules, err := m.natService.Setup(nat.Options{
		VPNNetwork:        config.Consumer.IPAddress,
		DNSIP:             dnsIP,
		ProviderExtIP:     net.ParseIP(m.outboundIP),
		EnableDNSRedirect: m.dnsOK,
		DNSPort:           m.dnsPort,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup NAT/firewall rules")
	}

	destroy := func() {
		if err := m.natService.Del(natRules); err != nil {
			log.Error().Err(err).Msg("Failed to delete NAT rules")
		}
		if err := conn.Stop(); err != nil {
			log.Error().Err(err).Msg("Failed to stop connection endpoint")
		}
	}

	return &session.ConfigParams{SessionServiceConfig: config, SessionDestroyCallback: destroy, TraversalParams: &traversalParams}, nil
}

func (m *Manager) startNewConnection() (wg.ConnectionEndpoint, error) {
	connEndpoint, err := m.connectionEndpointFactory()
	if err != nil {
		return nil, errors.Wrap(err, "could not run conn endpoint factory")
	}

	if err := connEndpoint.Start(wg.StartConfig{}); err != nil {
		return nil, errors.Wrap(err, "could not start provider wg connection endpoint")
	}
	return connEndpoint, nil
}

func (m *Manager) addConsumerPeer(conn wg.ConnectionEndpoint, consumerPort, providerPort int, peerPublicKey string) error {
	var peerEndpoint *net.UDPAddr
	if consumerPort > 0 {
		var err error
		peerEndpoint, err = net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", "127.0.0.1", providerPort))
		if err != nil {
			return errors.Wrap(err, "could not resolve new peer addr")
		}
	}
	peerOpts := wg.Peer{
		PublicKey:  peerPublicKey,
		Endpoint:   peerEndpoint,
		AllowedIPs: []string{"0.0.0.0/0", "::/0"},
	}
	return conn.AddPeer(conn.InterfaceName(), peerOpts)
}

func (m *Manager) addTraversalParams(config wg.ServiceConfig, traversalParams traversal.Params) (wg.ServiceConfig, error) {
	config.LocalPort = traversalParams.ConsumerPort
	config.RemotePort = traversalParams.ProviderPort

	// Provide new provider endpoint which points to providers NAT Proxy.
	newProviderEndpoint, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", config.Provider.Endpoint.IP, config.RemotePort))
	if err != nil {
		return wg.ServiceConfig{}, errors.Wrap(err, "could not resolve new provider endpoint")
	}
	config.Provider.Endpoint = *newProviderEndpoint

	return config, nil
}

func (m *Manager) newTraversalParams(natPingerEnabled bool, consumserConfig wg.ConsumerConfig) (traversal.Params, error) {
	params := traversal.Params{
		Cancel: make(chan struct{}),
	}

	if !natPingerEnabled {
		return params, nil
	}

	pp, err := m.natPingerPorts.Acquire()
	if err != nil {
		return params, errors.Wrap(err, "could not acquire NAT pinger provider port")
	}

	cp, err := m.natPingerPorts.Acquire()
	if err != nil {
		return params, errors.Wrap(err, "could not acquire NAT pinger consumer port")
	}

	params.ProviderPort = pp.Num()
	params.ConsumerPort = cp.Num()
	params.ProxyPortMappingKey = fmt.Sprintf("%s_%d", wg.ServiceType, params.ProviderPort)

	if consumserConfig.IP == "" {
		return params, errors.New("remote party does not support NAT Hole punching, public IP is missing")
	}
	params.ConsumerPublicIP = consumserConfig.IP

	return params, nil
}

// Serve starts service - does block
func (m *Manager) Serve(providerID identity.Identity) error {
	log.Info().Msg("Wireguard service started successfully now")

	// Start DNS proxy.
	m.dnsPort = 11253
	m.dnsOK = false
	m.dnsProxyMu.Lock()
	defer m.dnsProxyMu.Unlock()
	m.dnsProxy = dns.NewProxy("", m.dnsPort)
	if err := m.dnsProxy.Run(); err != nil {
		log.Warn().Err(err).Msg("Provider DNS will not be available")
	} else {
		// m.dnsProxy = dnsProxy
		m.dnsOK = true
	}

	<-m.done
	return nil
}

// Stop stops service.
func (m *Manager) Stop() error {
	close(m.done)

	// Stop DNS proxy.
	m.dnsProxyMu.Lock()
	defer m.dnsProxyMu.Unlock()
	if m.dnsProxy != nil {
		if err := m.dnsProxy.Stop(); err != nil {
			log.Error().Err(err).Msg("Failed to stop DNS server")
		}
	}

	log.Info().Msg("Wireguard service stopped")
	return nil
}

func (m *Manager) isBehindNAT() bool {
	return m.outboundIP != m.publicIP
}

func (m *Manager) portMappingFailed() bool {
	lastEvent := m.natEventGetter.LastEvent()
	if lastEvent == nil {
		return false
	}

	if lastEvent.Stage == traversal.StageName {
		return true
	}
	return lastEvent.Stage == mapping.StageName && !lastEvent.Successful
}
