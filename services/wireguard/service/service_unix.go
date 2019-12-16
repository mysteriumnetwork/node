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
	"net"
	"sync"

	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/dns"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/nat/traversal"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/utils/netutil"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// NewManager creates new instance of Wireguard service
func NewManager(
	ipResolver ip.Resolver,
	natService nat.NATService,
	portMap func(port int) (releasePortMapping func()),
	options Options,
	portSupplier port.ServicePortSupplier,
) *Manager {
	resourceAllocator := resources.NewAllocator(portSupplier, options.Subnet)
	return &Manager{
		natService: natService,
		ipResolver: ipResolver,

		connectionEndpointFactory: func() (wg.ConnectionEndpoint, error) {
			return endpoint.NewConnectionEndpoint(ipResolver, resourceAllocator, portMap, options.ConnectDelay)
		},
	}
}

// Manager represents an instance of Wireguard service
type Manager struct {
	wg         sync.WaitGroup
	natService nat.NATService

	connectionEndpointFactory func() (wg.ConnectionEndpoint, error)

	ipResolver ip.Resolver
}

// ProvideConfig provides the config for consumer
func (manager *Manager) ProvideConfig(sessionConfig json.RawMessage, traversalParams *traversal.Params) (*session.ConfigParams, error) {
	key := &wg.ConsumerConfig{}
	err := json.Unmarshal(sessionConfig, key)
	if err != nil {
		return nil, err
	}

	connectionEndpoint, err := manager.connectionEndpointFactory()
	if err != nil {
		return nil, err
	}

	if err := connectionEndpoint.Start(nil); err != nil {
		return nil, err
	}

	if err := connectionEndpoint.AddPeer(key.PublicKey, nil); err != nil {
		return nil, err
	}

	config, err := connectionEndpoint.Config()
	if err != nil {
		return nil, err
	}

	var dnsOK bool
	var dnsIP net.IP
	var dnsPort = 11253
	dnsProxy := dns.NewProxy("", dnsPort)
	if err := dnsProxy.Run(); err != nil {
		log.Warn().Err(err).Msg("Provider DNS will not be available")
	} else {
		dnsOK = true
		dnsIP = netutil.FirstIP(config.Consumer.IPAddress)
		config.Consumer.DNS = dnsIP.String()
	}

	outIP, err := manager.ipResolver.GetOutboundIP()
	if err != nil {
		return nil, err
	}

	natRules, err := manager.natService.Setup(nat.Options{
		VPNNetwork:        config.Consumer.IPAddress,
		ProviderExtIP:     outIP,
		EnableDNSRedirect: dnsOK,
		DNSIP:             dnsIP,
		DNSPort:           dnsPort,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup NAT/firewall rules")
	}

	destroy := func() {
		if err := dnsProxy.Stop(); err != nil {
			log.Error().Err(err).Msg("Failed to stop DNS server")
		}
		if err := manager.natService.Del(natRules); err != nil {
			log.Error().Err(err).Msg("Failed to delete NAT rules")
		}
		if err := connectionEndpoint.Stop(); err != nil {
			log.Error().Err(err).Msg("Failed to stop connection endpoint")
		}
	}

	return &session.ConfigParams{SessionServiceConfig: config, SessionDestroyCallback: destroy, TraversalParams: traversalParams}, nil
}

// Serve starts service - does block
func (manager *Manager) Serve(providerID identity.Identity) error {
	manager.wg.Add(1)
	log.Info().Msg("Wireguard service started successfully")

	manager.wg.Wait()
	return nil
}

// Stop stops service.
func (manager *Manager) Stop() error {
	manager.wg.Done()

	log.Info().Msg("Wireguard service stopped")
	return nil
}
