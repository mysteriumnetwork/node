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
	"sort"
	"sync"

	"github.com/mysteriumnetwork/node/core/service"
	"github.com/pkg/errors"
)

// MemoryBackend stores runtime service definitions and active/passive state in memory.
type MemoryBackend struct {
	mu       sync.RWMutex
	services map[string]Options
	active   map[string]struct{}
}

// NewMemoryBackend creates an in-memory runtime backend.
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		services: make(map[string]Options),
		active:   make(map[string]struct{}),
	}
}

// Create stores a runtime service definition as passive until it is started.
func (backend *MemoryBackend) Create(options Options) error {
	if options.Name == "" {
		return errors.New("runtime service name is required")
	}

	backend.mu.Lock()
	defer backend.mu.Unlock()

	backend.services[options.Name] = options
	return nil
}

// Delete removes a runtime service definition and clears any active state.
func (backend *MemoryBackend) Delete(name string) error {
	if name == "" {
		return errors.New("runtime service name is required")
	}

	backend.mu.Lock()
	defer backend.mu.Unlock()

	delete(backend.services, name)
	delete(backend.active, name)
	return nil
}

// Start marks a runtime service active.
func (backend *MemoryBackend) Start(_ *service.Instance, options Options) error {
	if options.Name == "" {
		return errors.New("runtime service name is required")
	}

	backend.mu.Lock()
	defer backend.mu.Unlock()

	backend.services[options.Name] = options
	backend.active[options.Name] = struct{}{}
	return nil
}

// Stop marks a runtime service passive.
func (backend *MemoryBackend) Stop(options Options) error {
	if options.Name == "" {
		return errors.New("runtime service name is required")
	}

	backend.mu.Lock()
	defer backend.mu.Unlock()

	delete(backend.active, options.Name)
	return nil
}

// List returns runtime services known to the backend with active/passive state.
func (backend *MemoryBackend) List() ([]ServiceInfo, error) {
	backend.mu.RLock()
	defer backend.mu.RUnlock()

	result := make([]ServiceInfo, 0, len(backend.services))
	for name, options := range backend.services {
		state := ServiceStatePassive
		if _, active := backend.active[name]; active {
			state = ServiceStateActive
		}

		result = append(result, ServiceInfo{
			Name:    name,
			State:   state,
			Options: options,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}
