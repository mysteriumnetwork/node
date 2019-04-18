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
	"sync"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/nat"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
	"github.com/mysteriumnetwork/node/session"
	"github.com/pkg/errors"
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
		natService:        natService,
		resourceAllocator: resourceAllocator,
		portMap:           portMap,
		ipResolver:        ipResolver,
		options:           options,
	}
}

// Manager represents an instance of Wireguard service
type Manager struct {
	wg         sync.WaitGroup
	natService nat.NATService

	connectionEndpoint wg.ConnectionEndpoint

	resourceAllocator *resources.Allocator

	ipResolver ip.Resolver
	portMap    func(port int) (releasePortMapping func())
	options    Options
}

// ProvideConfig provides the config for consumer
func (manager *Manager) ProvideConfig(publicKey json.RawMessage) (session.ServiceConfiguration, session.DestroyCallback, error) {
	key := &wg.ConsumerConfig{}
	err := json.Unmarshal(publicKey, key)
	if err != nil {
		return nil, nil, err
	}

	config, err := manager.connectionEndpoint.Config()
	if err != nil {
		return nil, nil, err
	}

	if err := manager.connectionEndpoint.AddPeer(key.PublicKey, nil, config.Consumer.IPAddress.IP.String()+"/32"); err != nil {
		return nil, nil, err
	}

	destroy := func() {
		if err := manager.resourceAllocator.ReleaseIPNet(config.Consumer.IPAddress); err != nil {
			log.Error(logPrefix, "failed to release IP network", err)
		}
		if err := manager.connectionEndpoint.RemovePeer(key.PublicKey); err != nil {
			log.Error(logPrefix, "failed to remove peer: ", key.PublicKey, err)
		}
	}

	return config, destroy, nil
}

// Serve starts service - does block
func (manager *Manager) Serve(providerID identity.Identity) error {
	manager.wg.Add(1)

	connectionEndpoint, err := endpoint.NewConnectionEndpoint(manager.ipResolver, manager.resourceAllocator, manager.portMap, manager.options.ConnectDelay)
	if err != nil {
		return err
	}

	if err := connectionEndpoint.Start(nil); err != nil {
		return err
	}

	outIP, err := manager.ipResolver.GetOutboundIP()
	if err != nil {
		return err
	}

	config, err := connectionEndpoint.Config()
	if err != nil {
		return err
	}

	if err := firewall.AddInboundRule("UDP", config.Provider.Endpoint.Port); err != nil {
		return errors.Wrap(err, "failed to add firewall rule")
	}
	defer func() {
		if err := firewall.RemoveInboundRule("UDP", config.Provider.Endpoint.Port); err != nil {
			log.Error(logPrefix, "Failed to delete firewall rule for Wireguard", err)
		}
	}()

	natRule := nat.RuleForwarding{SourceAddress: config.Consumer.IPAddress.String(), TargetIP: outIP}
	if err := manager.natService.Add(natRule); err != nil {
		return errors.Wrap(err, "failed to add NAT forwarding rule")
	}

	manager.connectionEndpoint = connectionEndpoint
	log.Info(logPrefix, "Wireguard service started successfully")

	manager.wg.Wait()
	return nil
}

// Stop stops service.
func (manager *Manager) Stop() error {
	manager.wg.Done()

	manager.connectionEndpoint.Stop()

	log.Info(logPrefix, "Wireguard service stopped")
	return nil
}
