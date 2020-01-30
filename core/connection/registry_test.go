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
	"testing"

	"github.com/stretchr/testify/assert"
)

var _ Creator = (&Registry{}).CreateConnection

var (
	serviceType = "serviceType"
)

func TestRegistry_Factory(t *testing.T) {
	registry := NewRegistry()
	assert.Len(t, registry.creators, 0)
}

func TestRegistry_Register(t *testing.T) {
	registry := Registry{
		creators: map[string]Factory{},
	}

	registry.Register(serviceType, func() (connection Connection, err error) {
		return &connectionMock{}, nil
	})
	assert.Len(t, registry.creators, 1)
}

func TestRegistry_CreateConnection_NonExisting(t *testing.T) {
	registry := &Registry{}

	connection, err := registry.CreateConnection(serviceType)
	assert.Equal(t, ErrUnsupportedServiceType, err)
	assert.Nil(t, connection)
}

func TestRegistry_CreateConnection_Existing(t *testing.T) {
	mock := &connectionMock{}
	registry := Registry{
		creators: map[string]Factory{
			"fake-service": func() (connection Connection, err error) {
				return mock, nil
			},
		},
	}

	connection, err := registry.CreateConnection("fake-service")
	assert.NoError(t, err)
	assert.Equal(t, mock, connection)
}
