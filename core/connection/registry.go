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

import "github.com/mysteriumnetwork/node/session"

// Registry holds of all plugable connections
type Registry struct {
	creators map[string]Creator
	acks     map[string]session.AckHandler
}

// NewRegistry creates registry of plugable connections
func NewRegistry() *Registry {
	return &Registry{
		creators: make(map[string]Creator),
		acks:     make(map[string]session.AckHandler),
	}
}

// Register new plugable connection
func (registry *Registry) Register(serviceType string, creator Creator) {
	registry.creators[serviceType] = creator
}

// CreateConnection create plugable connection
func (registry *Registry) CreateConnection(options ConnectOptions, stateChannel StateChannel, statisticsChannel StatisticsChannel) (Connection, error) {
	createConnection, exists := registry.creators[options.Proposal.ServiceType]
	if !exists {
		return nil, ErrUnsupportedServiceType
	}

	return createConnection(options, stateChannel, statisticsChannel)
}

// AddAck registers an ack handler for a service type
func (registry *Registry) AddAck(serviceType string, handler session.AckHandler) {
	registry.acks[serviceType] = handler
}

// GetAck returns the ack for the service type
func (registry *Registry) GetAck(serviceType string) (session.AckHandler, error) {
	ackHandler, exists := registry.acks[serviceType]
	if !exists {
		return nil, ErrAckNotRegistered
	}
	return ackHandler, nil
}
