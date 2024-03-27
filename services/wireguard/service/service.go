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
	"time"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/shaper"
	"github.com/mysteriumnetwork/node/dns"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/nat"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint"
	"github.com/mysteriumnetwork/node/services/wireguard/key"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
	"github.com/mysteriumnetwork/node/utils/netutil"
)

// NewManager creates new instance of Wireguard service
func NewManager(
	ipResolver ip.Resolver,
	country string,
	natService nat.NATService,
	eventBus eventbus.EventBus,
	trafficFirewall firewall.IncomingTrafficFirewall,
	resourcesAllocator *resources.Allocator,
	wgClientFactory *endpoint.WgClientFactory,
	dnsProxy *dns.Proxy,
) *Manager {
	return &Manager{
		done:               make(chan struct{}),
		resourcesAllocator: resourcesAllocator,
		ipResolver:         ipResolver,
		natService:         natService,
		eventBus:           eventBus,
		trafficFirewall:    trafficFirewall,
		dnsProxy:           dnsProxy,

		connEndpointFactory: func() (wg.ConnectionEndpoint, error) {
			return endpoint.NewConnectionEndpoint(resourcesAllocator, wgClientFactory)
		},
		country:        country,
		sessionCleanup: map[string]func(){},
	}
}

// Manager represents an instance of Wireguard service
type Manager struct {
	done        chan struct{}
	startStopMu sync.Mutex

	resourcesAllocator *resources.Allocator

	natService      nat.NATService
	eventBus        eventbus.EventBus
	trafficFirewall firewall.IncomingTrafficFirewall

	dnsProxy *dns.Proxy

	connEndpointFactory func() (wg.ConnectionEndpoint, error)

	ipResolver ip.Resolver

	serviceInstance  *service.Instance
	sessionCleanup   map[string]func()
	sessionCleanupMu sync.Mutex

	country    string
	outboundIP string
}

// ProvideConfig provides the config for consumer and handles new WireGuard connection.
func (m *Manager) ProvideConfig(sessionID string, sessionConfig json.RawMessage, remoteConn *net.UDPConn) (*service.ConfigParams, error) {
	log.Info().Msg("Accepting new WireGuard connection")
	consumerConfig := wg.ConsumerConfig{}
	err := json.Unmarshal(sessionConfig, &consumerConfig)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal wg consumer config")
	}

	remoteConn.Close()
	listenPort := remoteConn.LocalAddr().(*net.UDPAddr).Port
	providerConfig, err := m.createProviderConfig(listenPort, consumerConfig.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("could not create provider mode wg config: %w", err)
	}

	publicIP, err := m.ipResolver.GetPublicIP()
	if err != nil {
		return nil, errors.Wrap(err, "could not get public IP")
	}

	conn, err := m.startNewConnection(publicIP, providerConfig)
	if err != nil {
		return nil, errors.Wrap(err, "could not start new connection")
	}

	config, err := conn.Config()
	if err != nil {
		return nil, errors.Wrap(err, "could not get peer config")
	}

	var dnsIP net.IP
	var releaseTrafficFirewall firewall.IncomingRuleRemove
	if m.serviceInstance.PolicyProvider().HasDNSRules() {
		releaseTrafficFirewall, err = m.trafficFirewall.BlockIncomingTraffic(providerConfig.Subnet)
		if err != nil {
			return nil, errors.Wrap(err, "failed to enable traffic blocking")
		}
	}

	dnsIP = netutil.FirstIP(config.Consumer.IPAddress)
	config.Consumer.DNSIPs = dnsIP.String()

	natRules, err := m.natService.Setup(nat.Options{
		VPNNetwork:    config.Consumer.IPAddress,
		DNSIP:         dnsIP,
		ProviderExtIP: net.ParseIP(m.outboundIP),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup NAT/firewall rules")
	}

	statsPublisher := newStatsPublisher(m.eventBus, time.Second)
	go statsPublisher.start(sessionID, conn)

	ifaceName := conn.InterfaceName()
	s := shaper.New(m.eventBus)
	err = s.Start(ifaceName)
	if err != nil {
		log.Error().Err(err).Msg("Could not start traffic shaper")
	}

	destroy := func() {
		log.Info().Msgf("Cleaning up session %s", sessionID)
		m.sessionCleanupMu.Lock()
		defer m.sessionCleanupMu.Unlock()
		_, ok := m.sessionCleanup[sessionID]
		if !ok {
			log.Info().Msgf("Session '%s' was already cleaned up, returning without changes", sessionID)
			return
		}
		delete(m.sessionCleanup, sessionID)

		statsPublisher.stop()

		s.Clear(ifaceName)

		if releaseTrafficFirewall != nil {
			if err := releaseTrafficFirewall(); err != nil {
				log.Warn().Err(err).Msg("failed to disable traffic blocking")
			}
		}

		log.Trace().Msg("Deleting nat rules")
		if err := m.natService.Del(natRules); err != nil {
			log.Error().Err(err).Msg("Failed to delete NAT rules")
		}

		log.Trace().Msg("Stopping connection endpoint")
		if err := conn.Stop(); err != nil {
			log.Error().Err(err).Msg("Failed to stop connection endpoint")
		}

		if err := m.resourcesAllocator.ReleaseIPNet(providerConfig.Subnet); err != nil {
			log.Error().Err(err).Msg("Failed to release IP network")
		}
	}

	m.sessionCleanupMu.Lock()
	m.sessionCleanup[sessionID] = destroy
	m.sessionCleanupMu.Unlock()

	return &service.ConfigParams{SessionServiceConfig: config, SessionDestroyCallback: destroy}, nil
}

func (m *Manager) createProviderConfig(listenPort int, peerPublicKey string) (wgcfg.DeviceConfig, error) {
	network, err := m.resourcesAllocator.AllocateIPNet()
	if err != nil {
		return wgcfg.DeviceConfig{}, errors.Wrap(err, "could not allocate provider IP NET")
	}

	privateKey, err := key.GeneratePrivateKey()
	if err != nil {
		return wgcfg.DeviceConfig{}, fmt.Errorf("could not generate private key: %w", err)
	}

	return wgcfg.DeviceConfig{
		IfaceName:  "", // Interface name will be generated by connection endpoint.
		Subnet:     network,
		PrivateKey: privateKey,
		ListenPort: listenPort,
		DNSPort:    config.GetInt(config.FlagDNSListenPort),
		DNS:        nil,
		Peer: wgcfg.Peer{
			PublicKey: peerPublicKey,
			// Peer endpoint is set automatically by wg once client does handshake.
			Endpoint:               nil,
			AllowedIPs:             []string{"0.0.0.0/0", "::/0"},
			KeepAlivePeriodSeconds: 0,
		},
		ReplacePeers: true,
	}, nil
}

func (m *Manager) startNewConnection(publicIP string, config wgcfg.DeviceConfig) (wg.ConnectionEndpoint, error) {
	connEndpoint, err := m.connEndpointFactory()
	if err != nil {
		return nil, errors.Wrap(err, "could not run conn endpoint factory")
	}

	if err := connEndpoint.StartProviderMode(publicIP, config); err != nil {
		return nil, errors.Wrap(err, "could not start provider wg connection endpoint")
	}
	return connEndpoint, nil
}

// Serve starts service - does block
func (m *Manager) Serve(instance *service.Instance) error {
	log.Info().Msg("Wireguard: starting")
	m.startStopMu.Lock()
	m.serviceInstance = instance

	var err error
	m.outboundIP, err = m.ipResolver.GetOutboundIP()
	if err != nil {
		return errors.Wrap(err, "could not get outbound IP")
	}

	if err := m.dnsProxy.Run(); err != nil {
		log.Error().Err(err).Msg("Provider DNS will not be available")

		return err
	}

	m.startStopMu.Unlock()
	log.Info().Msg("Wireguard: started")
	<-m.done
	return nil
}

// Stop stops service.
func (m *Manager) Stop() error {
	log.Info().Msg("Wireguard: stopping")
	m.startStopMu.Lock()
	defer m.startStopMu.Unlock()

	cleanupWg := sync.WaitGroup{}

	// prevent concurrent iteration and write
	sessionCleanupCopy := make(map[string]func())
	if err := copier.Copy(&sessionCleanupCopy, m.sessionCleanup); err != nil {
		panic(err)
	}
	for k, v := range sessionCleanupCopy {
		cleanupWg.Add(1)
		go func(sessionID string, cleanup func()) {
			defer cleanupWg.Done()
			cleanup()
		}(k, v)
	}
	cleanupWg.Wait()

	// Stop DNS proxy.
	if m.dnsProxy != nil {
		if err := m.dnsProxy.Stop(); err != nil {
			log.Error().Err(err).Msg("Failed to stop DNS server")
		}
	}

	close(m.done)
	log.Info().Msg("Wireguard: stopped")
	return nil
}
