/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
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

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/p2p"
)

// ServiceState represents whether a runtime-defined service is active or passive.
type ServiceState string

const (
	ServiceStateActive  ServiceState = "active"
	ServiceStatePassive ServiceState = "passive"
)

// ServiceInfo describes a runtime-defined service known to the backend.
type ServiceInfo struct {
	Name    string       `json:"name"`
	State   ServiceState `json:"state"`
	Options Options      `json:"options,omitempty"`
}

// Backend represents the interface responsible for fetching, preparing and executing OCI-backed services.
type Backend interface {
	Create(options Options) error
	Delete(name string) error
	Start(instance *service.Instance, options Options) error
	Stop(options Options) error
	List() ([]ServiceInfo, error)
}

// Manager represents entrypoint for runtime-backed services.
type Manager struct {
	backend Backend
	options Options

	stopOnce sync.Once
	done     chan struct{}
}

// NewManager creates a new runtime service manager.
func NewManager(backend Backend, options Options) *Manager {
	return &Manager{
		backend: backend,
		options: options,
		done:    make(chan struct{}),
	}
}

// ProvideConfig provides the session configuration.
func (manager *Manager) ProvideConfig(_ string, _ json.RawMessage, _ p2p.ServiceConn) (*service.ConfigParams, error) {
	return &service.ConfigParams{}, nil
}

// Serve starts the service and blocks until Stop is called.
func (manager *Manager) Serve(instance *service.Instance) error {
	if manager.backend != nil {
		if err := manager.backend.Start(instance, manager.options); err != nil {
			return err
		}
	} else {
		log.Info().Str("artifact_address", manager.options.RootFS).Str("service_type", instance.Type).Msg("Runtime service started without backend")
	}

	<-manager.done
	return nil
}

// Stop stops the service.
func (manager *Manager) Stop() error {
	manager.stopOnce.Do(func() {
		close(manager.done)
	})

	if manager.backend != nil {
		return manager.backend.Stop(manager.options)
	}

	return nil
}

// List returns runtime-defined services known to the backend, including active and passive ones.
func (manager *Manager) List() ([]ServiceInfo, error) {
	if manager.backend == nil {
		return nil, nil
	}

	return manager.backend.List()
}
