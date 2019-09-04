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
	"github.com/mysteriumnetwork/node/utils"
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

	config.Consumer.DNS = utils.FirstIP(config.Consumer.IPAddress).String()
	dnsServer := dns.NewServer(
		net.JoinHostPort(config.Consumer.DNS, "53"),
		dns.ResolveViaConfigured(),
	)

	go func() {
		log.Info("starting DNS on: ", dnsServer.Addr)
		if err := dnsServer.Run(); err != nil {
			log.Error("failed to start DNS server: ", err)
		}
	}()

	outIP, err := manager.ipResolver.GetOutboundIPAsString()
	if err != nil {
		return nil, err
	}

	natRule := nat.RuleForwarding{SourceSubnet: config.Consumer.IPAddress.String(), TargetIP: outIP}
	if err := manager.natService.Add(natRule); err != nil {
		return nil, err
	}

	destroy := func() {
		if err := dnsServer.Stop(); err != nil {
			log.Error("failed to stop DNS server", err)
		}
		if err := manager.natService.Del(natRule); err != nil {
			log.Error("failed to delete NAT forwarding rule: ", err)
		}
		if err := connectionEndpoint.Stop(); err != nil {
			log.Error("failed to stop connection endpoint: ", err)
		}
	}

	return &session.ConfigParams{SessionServiceConfig: config, SessionDestroyCallback: destroy, TraversalParams: traversalParams}, nil
}

// Serve starts service - does block
func (manager *Manager) Serve(providerID identity.Identity) error {
	manager.wg.Add(1)
	log.Info("wireguard service started successfully")

	manager.wg.Wait()
	return nil
}

// Stop stops service.
func (manager *Manager) Stop() error {
	manager.wg.Done()

	log.Info("wireguard service stopped")
	return nil
}
