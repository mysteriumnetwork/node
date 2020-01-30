/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package connection

// Factory represents a connection constructor
type Factory func() (Connection, error)

// Registry holds of all plugable connections
type Registry struct {
	creators map[string]Factory
}

// NewRegistry creates registry of plugable connections
func NewRegistry() *Registry {
	return &Registry{
		creators: make(map[string]Factory),
	}
}

// Register new plugable connection
func (registry *Registry) Register(serviceType string, creator Factory) {
	registry.creators[serviceType] = creator
}

// CreateConnection create plugable connection
func (registry *Registry) CreateConnection(serviceType string) (Connection, error) {
	factory, exists := registry.creators[serviceType]
	if !exists {
		return nil, ErrUnsupportedServiceType
	}

	return factory()
}
