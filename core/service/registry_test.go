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

package service

import (
	"testing"

	"github.com/mysteriumnetwork/node/market"

	"github.com/stretchr/testify/assert"
)

var _ ServiceFactory = (&Registry{}).Create

var (
	proposalMock   = market.ServiceProposal{}
	serviceMock    = &serviceFake{}
	serviceFactory = func(options Options) (Service, market.ServiceProposal, error) {
		return serviceMock, proposalMock, nil
	}
)

func TestRegistry_Factory(t *testing.T) {
	registry := NewRegistry()
	assert.Len(t, registry.factories, 0)
}

func TestRegistry_Register(t *testing.T) {
	registry := Registry{
		factories: map[string]ServiceFactory{},
	}

	registry.Register("any", serviceFactory)
	assert.Len(t, registry.factories, 1)
}

func TestRegistry_Create_NonExisting(t *testing.T) {
	registry := &Registry{}

	service, proposal, err := registry.Create(Options{})
	assert.Equal(t, ErrUnsupportedServiceType, err)
	assert.Nil(t, service)
	assert.Equal(t, proposalMock, proposal)
}

func TestRegistry_Create_Existing(t *testing.T) {
	registry := Registry{
		factories: map[string]ServiceFactory{
			"fake-service": serviceFactory,
		},
	}

	service, proposal, err := registry.Create(Options{
		Type: "fake-service",
	})
	assert.NoError(t, err)
	assert.Equal(t, serviceMock, service)
	assert.Equal(t, proposalMock, proposal)
}
