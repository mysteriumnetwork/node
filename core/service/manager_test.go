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
	"time"

	"github.com/mysteriumnetwork/node/core/policy/localcopy"
	"github.com/mysteriumnetwork/node/core/policy/requested"

	"github.com/mysteriumnetwork/node/core/location/locationstate"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/node/utils/netutil"
	"github.com/stretchr/testify/assert"
)

var (
	serviceType        = "the-very-awesome-test-service-type"
	mockPolicyOracle   = localcopy.NewOracle(requests.NewHTTPClient("0.0.0.0", requests.DefaultTimeout), "http://policy.localhost/", 1*time.Minute, true)
	mockPolicyProvider = requested.NewRequestedProvider(requests.NewHTTPClient("0.0.0.0", requests.DefaultTimeout), "http://policy.localhost/")
)

func init() {
	netutil.LogNetworkStats = func() {}
}

func TestManager_StartRemovesServiceFromPoolIfServiceCrashes(t *testing.T) {
	registry := NewRegistry()
	mockCopy := *serviceMock
	mockCopy.onStartReturnError = errors.New("some error")
	registry.Register(serviceType, func(serviceType string, options Options) (Service, error) {
		return &mockCopy, nil
	})

	discovery := mockDiscovery{}
	discoveryFactory := MockDiscoveryFactoryFunc(&discovery)
	manager := NewManager(
		registry,
		discoveryFactory,
		mocks.NewEventBus(),
		mockPolicyOracle,
		mockPolicyProvider,
		&mockP2PListener{}, nil, nil, mockLocationResolver{},
	)
	_, err := manager.Start(identity.FromAddress(proposalMock.ProviderID), serviceType, nil, struct{}{})
	assert.Nil(t, err)

	discovery.Wait()
	assert.Len(t, manager.servicePool.List(), 0)
}

func TestManager_StartDoesNotCrashIfStoppedByUser(t *testing.T) {
	registry := NewRegistry()
	mockCopy := *serviceMock
	mockCopy.mockProcess = make(chan struct{})
	registry.Register(serviceType, func(serviceType string, options Options) (Service, error) {
		return &mockCopy, nil
	})

	discovery := mockDiscovery{}
	discoveryFactory := MockDiscoveryFactoryFunc(&discovery)
	manager := NewManager(
		registry,
		discoveryFactory,
		mocks.NewEventBus(),
		mockPolicyOracle,
		mockPolicyProvider,
		&mockP2PListener{}, nil, nil,
		mockLocationResolver{},
	)
	id, err := manager.Start(identity.FromAddress(proposalMock.ProviderID), serviceType, nil, struct{}{})
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
	registry.Register(serviceType, func(serviceType string, options Options) (Service, error) {
		return &mockCopy, nil
	})

	discovery := mockDiscovery{}
	discoveryFactory := MockDiscoveryFactoryFunc(&discovery)
	eventBus := &mockPublisher{}
	manager := NewManager(
		registry,
		discoveryFactory,
		eventBus,
		mockPolicyOracle,
		mockPolicyProvider,
		&mockP2PListener{}, nil, nil,
		mockLocationResolver{},
	)

	id, err := manager.Start(identity.FromAddress(proposalMock.ProviderID), serviceType, nil, struct{}{})
	assert.NoError(t, err)

	services := manager.servicePool.List()

	var serviceID ID
	for k := range services {
		serviceID = services[k].ID
	}

	err = manager.Stop(id)
	assert.NoError(t, err)

	time.Sleep(time.Millisecond * 30)

	eventBus.lock.Lock()
	defer eventBus.lock.Unlock()

	assert.Equal(t, servicestate.AppTopicServiceStatus, eventBus.publishedTopic)

	var matchFound bool
	expectedPayload := servicestate.AppEventServiceStatus{ID: string(serviceID), ProviderID: "", Type: "", Status: "NotRunning"}
	for i := range eventBus.publishedData {
		e, ok := eventBus.publishedData[i].(servicestate.AppEventServiceStatus)
		if !ok {
			continue
		}
		if e.Status == expectedPayload.Status && e.ID == expectedPayload.ID {
			matchFound = true
			break
		}
	}
	assert.True(t, matchFound)
}

// A runtime service restored on boot and the same service started from the
// configured list race each other, so rejecting the second start has to be
// decided by the manager rather than by a check the caller made earlier.
func TestManager_StartRefusesConcurrentDuplicateOfSameService(t *testing.T) {
	registry := NewRegistry()
	registry.Register(serviceType, func(serviceType string, options Options) (Service, error) {
		return &serviceFake{mockProcess: make(chan struct{})}, nil
	})

	discovery := mockDiscovery{}
	manager := NewManager(
		registry,
		MockDiscoveryFactoryFunc(&discovery),
		mocks.NewEventBus(),
		mockPolicyOracle,
		mockPolicyProvider,
		&mockP2PListener{}, nil, nil,
		mockLocationResolver{},
	)

	providerID := identity.FromAddress(proposalMock.ProviderID)
	const starts = 8
	results := make(chan error, starts)
	begin := make(chan struct{})
	for i := 0; i < starts; i++ {
		go func() {
			<-begin
			_, err := manager.Start(providerID, serviceType, nil, struct{}{})
			results <- err
		}()
	}
	close(begin)

	started := 0
	for i := 0; i < starts; i++ {
		err := <-results
		if err == nil {
			started++
			continue
		}
		assert.ErrorIs(t, err, ErrorAlreadyRunning)
	}

	assert.Equal(t, 1, started)
	assert.Len(t, manager.servicePool.List(), 1)
}

// A refused start must not leave a claim behind, or the service could never be
// started again after it is stopped.
func TestManager_StartIsPossibleAgainAfterStop(t *testing.T) {
	registry := NewRegistry()
	registry.Register(serviceType, func(serviceType string, options Options) (Service, error) {
		return &serviceFake{mockProcess: make(chan struct{})}, nil
	})

	discovery := mockDiscovery{}
	manager := NewManager(
		registry,
		MockDiscoveryFactoryFunc(&discovery),
		mocks.NewEventBus(),
		mockPolicyOracle,
		mockPolicyProvider,
		&mockP2PListener{}, nil, nil,
		mockLocationResolver{},
	)

	providerID := identity.FromAddress(proposalMock.ProviderID)
	id, err := manager.Start(providerID, serviceType, nil, struct{}{})
	assert.NoError(t, err)

	_, err = manager.Start(providerID, serviceType, nil, struct{}{})
	assert.ErrorIs(t, err, ErrorAlreadyRunning)

	assert.NoError(t, manager.Stop(id))
	discovery.Wait()

	_, err = manager.Start(providerID, serviceType, nil, struct{}{})
	assert.NoError(t, err)
}

type mockP2PListener struct {
}

func (m mockP2PListener) GetContact() market.Contact {
	return market.Contact{}
}

func (m mockP2PListener) Listen(_ identity.Identity, serviceType string, channelHandler func(ch p2p.Channel)) (func(), error) {
	return func() {}, nil
}

type mockLocationResolver struct {
}

func (m mockLocationResolver) DetectLocation() (locationstate.Location, error) {
	return locationstate.Location{}, nil
}
