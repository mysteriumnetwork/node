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
	"errors"
	"testing"

	"github.com/mysteriumnetwork/node/market"
	"github.com/stretchr/testify/assert"
)

var _ ServiceFactory = (&Registry{}).Create

var (
	proposalMock = market.ServiceProposal{}
	serviceMock  = &serviceFake{}
)

func TestRegistry_Factory(t *testing.T) {
	registry := NewRegistry()
	assert.Len(t, registry.factories, 0)
}

func TestRegistry_Register(t *testing.T) {
	registry := mockRegistryEmpty()

	registry.Register(
		"any",
		func(serviceType string, options Options) (Service, market.ServiceProposal, error) {
			return serviceMock, proposalMock, nil
		},
	)
	assert.Len(t, registry.factories, 1)
}

func TestRegistry_Create_NonExisting(t *testing.T) {
	registry := mockRegistryEmpty()

	service, proposal, err := registry.Create("missing-service", nil)
	assert.Nil(t, service)
	assert.Equal(t, proposalMock, proposal)
	assert.Equal(t, ErrUnsupportedServiceType, err)
}

func TestRegistry_Create_Existing(t *testing.T) {
	registry := mockRegistryWith(
		"fake-service",
		func(_ string, options Options) (Service, market.ServiceProposal, error) {
			return serviceMock, proposalMock, nil
		},
	)

	service, proposal, err := registry.Create("fake-service", nil)
	assert.Equal(t, serviceMock, service)
	assert.Equal(t, proposalMock, proposal)
	assert.NoError(t, err)
}

func TestRegistry_Create_BubblesErrors(t *testing.T) {
	fakeErr := errors.New("I am broken")
	registry := mockRegistryWith(
		"broken-service",
		func(_ string, options Options) (Service, market.ServiceProposal, error) {
			return nil, proposalMock, fakeErr
		},
	)

	service, proposal, err := registry.Create("broken-service", nil)
	assert.Nil(t, service)
	assert.Equal(t, proposalMock, proposal)
	assert.Exactly(t, fakeErr, err)
}

func mockRegistryEmpty() *Registry {
	return &Registry{
		factories: map[string]ServiceFactory{},
	}
}

func mockRegistryWith(serviceType string, serviceFactory ServiceFactory) *Registry {
	return &Registry{
		factories: map[string]ServiceFactory{
			serviceType: serviceFactory,
		},
	}
}
