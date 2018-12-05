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

import (
	"encoding/json"
	"testing"

	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/session"
	"github.com/stretchr/testify/assert"
)

var _ Creator = (&Registry{}).CreateConnection

var (
	connectionMock    = &connectionFake{}
	connectionFactory = func(connectionParams ConnectOptions, stateChannel StateChannel, statisticsChannel StatisticsChannel) (Connection, error) {
		return connectionMock, nil
	}
	mockAckHandler = func(sessionResponse session.SessionDto, ackSend func(payload interface{}) error) (json.RawMessage, error) {
		return nil, nil
	}
	serviceType = "serviceType"
)

func TestRegistry_Factory(t *testing.T) {
	registry := NewRegistry()
	assert.Len(t, registry.creators, 0)
}

func TestRegistry_Register(t *testing.T) {
	registry := Registry{
		creators: map[string]Creator{},
	}

	registry.Register(serviceType, connectionFactory)
	assert.Len(t, registry.creators, 1)
}

func TestRegistry_CreateConnection_NonExisting(t *testing.T) {
	registry := &Registry{}

	connection, err := registry.CreateConnection(ConnectOptions{}, make(chan State), make(chan consumer.SessionStatistics))
	assert.Equal(t, ErrUnsupportedServiceType, err)
	assert.Nil(t, connection)
}

func TestRegistry_CreateConnection_Existing(t *testing.T) {
	connectOptions := ConnectOptions{
		Proposal: market.ServiceProposal{ServiceType: "fake-service"},
	}

	registry := Registry{
		creators: map[string]Creator{
			"fake-service": connectionFactory,
		},
	}

	connection, err := registry.CreateConnection(connectOptions, make(chan State), make(chan consumer.SessionStatistics))
	assert.NoError(t, err)
	assert.Equal(t, connectionMock, connection)
}

func TestRegistryAddAck(t *testing.T) {
	registry := Registry{
		acks: map[string]session.AckHandler{},
	}

	registry.AddAck(serviceType, mockAckHandler)
	assert.Len(t, registry.acks, 1)
}

func TestRegistryGetAckExists(t *testing.T) {
	registry := Registry{
		acks: map[string]session.AckHandler{},
	}
	serviceType := serviceType

	registry.AddAck(serviceType, mockAckHandler)
	handler, err := registry.GetAck(serviceType)
	assert.Nil(t, err)
	assert.NotNil(t, handler)
}

func TestRegistryGetAckNonExisting(t *testing.T) {
	registry := Registry{
		acks: map[string]session.AckHandler{},
	}
	serviceType := serviceType

	registry.AddAck("any", mockAckHandler)
	handler, err := registry.GetAck(serviceType)
	assert.Nil(t, handler)
	assert.Equal(t, ErrAckNotRegistered, err)
}
