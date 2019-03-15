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
	"github.com/mysteriumnetwork/node/core/location"
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
	location location.ServiceLocationInfo,
	natService nat.NATService,
	portMap func(port int) (releasePortMapping func()),
	options Options) *Manager {

	resourceAllocator := resources.NewAllocator()
	return &Manager{
		natService:        natService,
		resourceAllocator: &resourceAllocator,
		portMap:           portMap,
		location:          location,
		options:           options,
	}
}

// Manager represents an instance of Wireguard service
type Manager struct {
	wg         sync.WaitGroup
	natService nat.NATService

	connectionEndpoint wg.ConnectionEndpoint

	resourceAllocator *resources.Allocator

	location location.ServiceLocationInfo
	portMap  func(port int) (releasePortMapping func())
	options  Options

	mu   sync.Mutex // TODO this is a temporary solution to cleanup oldest used wireguard resources.
	list []*func()  // TODO it should be removed once payment bases session cleanup implemented.
}

// ProvideConfig provides the config for consumer
func (manager *Manager) ProvideConfig(publicKey json.RawMessage) (session.ServiceConfiguration, session.DestroyCallback, error) {
	key := &wg.ConsumerConfig{}
	err := json.Unmarshal(publicKey, key)
	if err != nil {
		return nil, nil, err
	}

	manager.cleanOldEndpoints()

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
	}

	return config, manager.once(destroy), nil
}

// Serve starts service - does block
func (manager *Manager) Serve(providerID identity.Identity) error {
	manager.wg.Add(1)

	connectionEndpoint, err := endpoint.NewConnectionEndpoint(manager.location, manager.resourceAllocator, manager.portMap, manager.options.ConnectDelay)
	if err != nil {
		return err
	}

	if err := connectionEndpoint.Start(nil); err != nil {
		return err
	}

	config, err := connectionEndpoint.Config()
	if err != nil {
		return err
	}

	natRule := nat.RuleForwarding{SourceAddress: config.Consumer.IPAddress.String(), TargetIP: manager.location.OutIP}
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
