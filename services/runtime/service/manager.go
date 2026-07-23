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
	"strings"
	"sync"

	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/p2p"
	runtime_service "github.com/mysteriumnetwork/node/services/runtime"
	runtime_capabilities "github.com/mysteriumnetwork/runtime/capabilities"
	runtime_lib_service "github.com/mysteriumnetwork/runtime/service"
)

// ServiceState represents whether a runtime-defined service is active or passive.
type ServiceState = runtime_lib_service.ServiceState

const (
	ServiceStateActive  ServiceState = runtime_lib_service.ServiceStateActive
	ServiceStatePassive ServiceState = runtime_lib_service.ServiceStatePassive
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
	Start(options Options) error
	Stop(options Options) error
	List() ([]ServiceInfo, error)
	Capabilities() (runtime_capabilities.RuntimeCapabilities, runtime_capabilities.DetailedCapabilities)
}

// Manager represents entrypoint for runtime-backed services.
type Manager struct {
	backend        Backend
	options        Options
	networkService service.Service

	stateMu       sync.Mutex
	activeOptions Options
	isBaseRuntime bool

	stopOnce sync.Once
	done     chan struct{}
}

// NewManager creates a new runtime service manager.
func NewManager(backend Backend, options Options, networkService service.Service) *Manager {
	return &Manager{
		backend:        backend,
		options:        options,
		networkService: networkService,
		done:           make(chan struct{}),
	}
}

// ProvideConfig provides the session configuration.
func (manager *Manager) ProvideConfig(sessionID string, sessionConfig json.RawMessage, conn p2p.ServiceConn) (*service.ConfigParams, error) {
	if manager.networkService != nil {
		return manager.networkService.ProvideConfig(sessionID, sessionConfig, conn)
	}

	return &service.ConfigParams{}, nil
}

// Serve starts the service and blocks until Stop is called.
func (manager *Manager) Serve(instance *service.Instance) error {
	managedOptions := manager.options
	isBaseRuntime := false
	if instance != nil {
		if instance.Type == runtime_service.ServiceType {
			// Base runtime service is a control host and should not create a workload container.
			isBaseRuntime = true
			manager.stateMu.Lock()
			manager.activeOptions = Options{}
			manager.isBaseRuntime = true
			manager.stateMu.Unlock()
			<-manager.done
			return nil
		}

		if managedOptions.Name == "" && strings.HasPrefix(instance.Type, runtime_service.ServiceType+".") {
			managedOptions.Name = instance.Type
		}
	}

	if manager.backend != nil {
		if err := manager.backend.Start(managedOptions); err != nil {
			return err
		}
	}

	manager.stateMu.Lock()
	manager.activeOptions = managedOptions
	manager.isBaseRuntime = isBaseRuntime
	manager.stateMu.Unlock()

	if manager.networkService != nil {
		return manager.networkService.Serve(instance)
	}

	<-manager.done
	return nil
}

// Stop stops the service.
func (manager *Manager) Stop() error {
	manager.stopOnce.Do(func() {
		close(manager.done)
	})

	manager.stateMu.Lock()
	activeOptions := manager.activeOptions
	isBaseRuntime := manager.isBaseRuntime
	manager.stateMu.Unlock()

	var stopErr error
	if manager.networkService != nil && !isBaseRuntime {
		if err := manager.networkService.Stop(); err != nil && stopErr == nil {
			stopErr = err
		}
	}

	if manager.backend != nil {
		if activeOptions.Name != "" {
			if err := manager.backend.Stop(activeOptions); err != nil && stopErr == nil {
				stopErr = err
			}
		}
	}

	return stopErr
}

// List returns runtime-defined services known to the backend, including active and passive ones.
func (manager *Manager) List() ([]ServiceInfo, error) {
	if manager.backend == nil {
		return nil, nil
	}

	return manager.backend.List()
}

// Capabilities returns the detected runtime capabilities of the underlying execution host.
func (manager *Manager) Capabilities() (runtime_capabilities.RuntimeCapabilities, runtime_capabilities.DetailedCapabilities) {
	if manager.backend == nil {
		return runtime_capabilities.Detect()
	}
	return manager.backend.Capabilities()
}
