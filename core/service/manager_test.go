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
	"errors"
	"testing"

	"github.com/asaskevich/EventBus"
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
)

var (
	serviceType = "the-very-awesome-test-service-type"
)

func TestManager_StartRemovesServiceFromPoolIfServiceCrashes(t *testing.T) {
	registry := NewRegistry()
	mockCopy := *serviceMock
	mockCopy.onStartReturnError = errors.New("some error")
	registry.Register(serviceType, func(options Options) (Service, market.ServiceProposal, error) {
		return &mockCopy, proposalMock, nil
	})

	discovery := mockDiscovery{}
	discoveryFactory := MockDiscoveryFactoryFunc(&discovery)
	manager := NewManager(
		registry,
		MockDialogWaiterFactory,
		MockDialogHandlerFactory,
		discoveryFactory,
		&MockNATPinger{},
		EventBus.New(),
	)
	_, err := manager.Start(identity.FromAddress(proposalMock.ProviderID), serviceType, struct{}{})
	assert.Nil(t, err)

	discovery.Wait()
	assert.Len(t, manager.servicePool.List(), 0)
}

func TestManager_StartDoesNotCrashIfStoppedByUser(t *testing.T) {
	registry := NewRegistry()
	mockCopy := *serviceMock
	mockCopy.mockProcess = make(chan struct{})
	registry.Register(serviceType, func(options Options) (Service, market.ServiceProposal, error) {
		return &mockCopy, proposalMock, nil
	})

	discovery := mockDiscovery{}
	discoveryFactory := MockDiscoveryFactoryFunc(&discovery)
	manager := NewManager(
		registry,
		MockDialogWaiterFactory,
		MockDialogHandlerFactory,
		discoveryFactory,
		&MockNATPinger{},
		EventBus.New(),
	)
	id, err := manager.Start(identity.FromAddress(proposalMock.ProviderID), serviceType, struct{}{})
	assert.Nil(t, err)
	err = manager.Stop(id)
	assert.Nil(t, err)
	discovery.Wait()
	assert.Len(t, manager.servicePool.List(), 0)
}

func TestManager_StopSendsEvent_SucceedsAndPublishesEvent(t *testing.T) {
	registry := NewRegistry()
	mockCopy := *serviceMock
	mockCopy.mockProcess = make(chan struct{})
	registry.Register(serviceType, func(options Options) (Service, market.ServiceProposal, error) {
		return &mockCopy, proposalMock, nil
	})

	discovery := mockDiscovery{}
	discoveryFactory := MockDiscoveryFactoryFunc(&discovery)
	eventBus := EventBus.New()
	manager := NewManager(
		registry,
		MockDialogWaiterFactory,
		MockDialogHandlerFactory,
		discoveryFactory,
		&MockNATPinger{},
		eventBus,
	)

	id, err := manager.Start(identity.FromAddress(proposalMock.ProviderID), serviceType, struct{}{})
	assert.NoError(t, err)

	var invokedInstance *Instance
	f := func(instance *Instance) {
		invokedInstance = instance
	}
	err = eventBus.Subscribe(StopTopic, f)
	assert.NoError(t, err)

	err = manager.Stop(id)
	assert.NoError(t, err)

	assert.Equal(t, &mockCopy, invokedInstance.service)
}
