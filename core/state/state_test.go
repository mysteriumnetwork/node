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

package state

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/state/event"
	"github.com/mysteriumnetwork/node/nat"
	natEvent "github.com/mysteriumnetwork/node/nat/event"
	"github.com/mysteriumnetwork/node/session"
	sessionEvent "github.com/mysteriumnetwork/node/session/event"
	"github.com/stretchr/testify/assert"
)

type debounceTester struct {
	numInteractions int
	lock            sync.Mutex
}

type interactionCounter interface {
	interactions() int
}

func (dt *debounceTester) do(interface{}) {
	dt.lock.Lock()
	dt.numInteractions++
	dt.lock.Unlock()
}

func (dt *debounceTester) interactions() int {
	dt.lock.Lock()
	defer dt.lock.Unlock()
	return dt.numInteractions
}

func Test_Debounce_CallsOnceInInterval(t *testing.T) {
	dt := &debounceTester{}
	duration := time.Millisecond * 10
	f := debounce(dt.do, duration)
	for i := 1; i < 10; i++ {
		f(struct{}{})
	}
	assert.Eventually(t, interacted(dt, 1), 2*time.Second, 10*time.Millisecond)
}

var mockNATStatus = nat.Status{
	Status: "status",
	Error:  errors.New("err"),
}

type natStatusProviderMock struct {
	statusToReturn  nat.Status
	numInteractions int
	lock            sync.Mutex
}

func (nspm *natStatusProviderMock) Status() nat.Status {
	nspm.lock.Lock()
	defer nspm.lock.Unlock()
	nspm.numInteractions++
	return nspm.statusToReturn
}

func (nspm *natStatusProviderMock) interactions() int {
	nspm.lock.Lock()
	defer nspm.lock.Unlock()
	return nspm.numInteractions
}

func (nspm *natStatusProviderMock) ConsumeNATEvent(event natEvent.Event) {}

type mockPublisher struct {
	lock           sync.Mutex
	publishedTopic string
	publishedData  interface{}
}

func (mp *mockPublisher) Publish(topic string, data interface{}) {
	mp.lock.Lock()
	defer mp.lock.Unlock()
	mp.publishedData = data
	mp.publishedTopic = topic
}

type serviceListerMock struct {
	lock             sync.Mutex
	numInteractions  int
	servicesToReturn map[service.ID]*service.Instance
}

func (slm *serviceListerMock) interactions() int {
	slm.lock.Lock()
	defer slm.lock.Unlock()
	return slm.numInteractions
}

func (slm *serviceListerMock) List() map[service.ID]*service.Instance {
	slm.lock.Lock()
	defer slm.lock.Unlock()
	slm.numInteractions++
	return slm.servicesToReturn
}

type serviceSessionStorageMock struct {
	numInteractions  int
	sessionsToReturn []session.Session
	lock             sync.Mutex
}

func (sssm *serviceSessionStorageMock) GetAll() []session.Session {
	sssm.lock.Lock()
	defer sssm.lock.Unlock()
	sssm.numInteractions++
	return sssm.sessionsToReturn
}

func (sssm *serviceSessionStorageMock) interactions() int {
	sssm.lock.Lock()
	defer sssm.lock.Unlock()
	return sssm.numInteractions
}

func Test_ConsumesNATEvents(t *testing.T) {
	natProvider := &natStatusProviderMock{
		statusToReturn: mockNATStatus,
	}
	publisher := &mockPublisher{}
	sl := &serviceListerMock{}
	sessionStorage := &serviceSessionStorageMock{}

	duration := time.Millisecond * 3
	keeper := NewKeeper(natProvider, publisher, sl, sessionStorage, duration)

	for i := 0; i < 5; i++ {
		// shoot a few events to see if we'll debounce
		keeper.ConsumeNATEvent(natEvent.Event{
			Stage:      "booster separation",
			Successful: false,
			Error:      errors.New("explosive bolts failed"),
		})
	}

	assert.Eventually(t, interacted(natProvider, 1), 2*time.Second, 10*time.Millisecond)

	assert.Equal(t, natProvider.statusToReturn.Error.Error(), keeper.GetState().NATStatus.Error)
	assert.Equal(t, natProvider.statusToReturn.Status, keeper.GetState().NATStatus.Status)
}

func Test_ConsumesSessionEvents(t *testing.T) {
	expected := session.Session{}

	natProvider := &natStatusProviderMock{
		statusToReturn: mockNATStatus,
	}
	publisher := &mockPublisher{}
	sl := &serviceListerMock{}
	sessionStorage := &serviceSessionStorageMock{
		sessionsToReturn: []session.Session{
			expected,
		},
	}

	duration := time.Millisecond * 3
	keeper := NewKeeper(natProvider, publisher, sl, sessionStorage, duration)

	for i := 0; i < 5; i++ {
		// shoot a few events to see if we'll debounce
		keeper.ConsumeSessionStateEvent(sessionEvent.Payload{})
	}

	assert.Eventually(t, interacted(sessionStorage, 1), 2*time.Second, 10*time.Millisecond)

	assert.Equal(t, string(expected.ID), keeper.GetState().Sessions[0].ID)
	assert.Equal(t, expected.ConsumerID.Address, keeper.GetState().Sessions[0].ConsumerID)
	assert.True(t, expected.CreatedAt.Equal(keeper.GetState().Sessions[0].CreatedAt))
}

func Test_ConsumesSessionAcknowledgeEvents(t *testing.T) {
	myID := "test"
	expected := event.ServiceSession{
		ID:        myID,
		ServiceID: myID,
	}

	natProvider := &natStatusProviderMock{
		statusToReturn: mockNATStatus,
	}
	publisher := &mockPublisher{}
	sl := &serviceListerMock{}
	sessionStorage := &serviceSessionStorageMock{
		sessionsToReturn: []session.Session{},
	}

	duration := time.Millisecond * 3
	keeper := NewKeeper(natProvider, publisher, sl, sessionStorage, duration)
	keeper.state.Services = []event.ServiceInfo{
		{ID: myID},
	}
	keeper.state.Sessions = []event.ServiceSession{
		expected,
	}

	keeper.ConsumeSessionStateEvent(sessionEvent.Payload{
		Action: sessionEvent.Acknowledged,
		ID:     string(expected.ID),
	})

	assert.Equal(t, 1, keeper.state.Services[0].ConnectionStatistics.Successful)
}

func Test_ConsumesServiceEvents(t *testing.T) {
	expected := service.Instance{}
	var id service.ID

	natProvider := &natStatusProviderMock{
		statusToReturn: mockNATStatus,
	}
	publisher := &mockPublisher{}
	sl := &serviceListerMock{
		servicesToReturn: map[service.ID]*service.Instance{
			id: &expected,
		},
	}

	sessionStorage := &serviceSessionStorageMock{}

	duration := time.Millisecond * 3
	keeper := NewKeeper(natProvider, publisher, sl, sessionStorage, duration)

	for i := 0; i < 5; i++ {
		// shoot a few events to see if we'll debounce
		keeper.ConsumeServiceStateEvent(service.EventPayload{})
	}

	assert.Eventually(t, interacted(sl, 1), 2*time.Second, 10*time.Millisecond)

	actual := keeper.GetState().Services[0]
	assert.Equal(t, string(id), actual.ID)
	assert.Equal(t, expected.Proposal().ServiceType, actual.Type)
	assert.Equal(t, expected.Proposal().ProviderID, actual.ProviderID)
	assert.Equal(t, expected.Options(), actual.Options)
	assert.Equal(t, string(expected.State()), actual.Status)
	assert.EqualValues(t, expected.Proposal(), actual.Proposal)
}

func Test_getServiceByID(t *testing.T) {

	natProvider := &natStatusProviderMock{
		statusToReturn: mockNATStatus,
	}
	publisher := &mockPublisher{}
	sl := &serviceListerMock{
		servicesToReturn: map[service.ID]*service.Instance{},
	}

	sessionStorage := &serviceSessionStorageMock{}

	duration := time.Millisecond * 3
	keeper := NewKeeper(natProvider, publisher, sl, sessionStorage, duration)
	myID := "test"
	keeper.state.Services = []event.ServiceInfo{
		{ID: myID},
		{ID: "mock"},
	}

	s, found := keeper.getServiceByID(myID)
	assert.True(t, found)

	assert.EqualValues(t, keeper.state.Services[0], s)

	_, found = keeper.getServiceByID("something else")
	assert.False(t, found)
}

func Test_getSessionByID(t *testing.T) {

	natProvider := &natStatusProviderMock{
		statusToReturn: mockNATStatus,
	}
	publisher := &mockPublisher{}
	sl := &serviceListerMock{
		servicesToReturn: map[service.ID]*service.Instance{},
	}

	sessionStorage := &serviceSessionStorageMock{}

	duration := time.Millisecond * 3
	keeper := NewKeeper(natProvider, publisher, sl, sessionStorage, duration)
	myID := "test"
	keeper.state.Sessions = []event.ServiceSession{
		{ID: myID},
		{ID: "mock"},
	}

	s, found := keeper.getSessionByID(myID)
	assert.True(t, found)

	assert.EqualValues(t, keeper.state.Sessions[0], s)

	_, found = keeper.getServiceByID("something else")
	assert.False(t, found)
}

func Test_incrementConnectionCount(t *testing.T) {
	expected := service.Instance{}
	var id service.ID

	natProvider := &natStatusProviderMock{
		statusToReturn: mockNATStatus,
	}
	publisher := &mockPublisher{}
	sl := &serviceListerMock{
		servicesToReturn: map[service.ID]*service.Instance{
			id: &expected,
		},
	}

	sessionStorage := &serviceSessionStorageMock{}

	duration := time.Millisecond * 3
	keeper := NewKeeper(natProvider, publisher, sl, sessionStorage, duration)
	myID := "test"
	keeper.state.Services = []event.ServiceInfo{
		{ID: myID},
		{ID: "mock"},
	}

	keeper.incrementConnectCount(myID, false)
	s, found := keeper.getServiceByID(myID)
	assert.True(t, found)

	assert.Equal(t, 1, s.ConnectionStatistics.Attempted)
	assert.Equal(t, 0, s.ConnectionStatistics.Successful)

	keeper.incrementConnectCount(myID, true)
	s, found = keeper.getServiceByID(myID)
	assert.True(t, found)

	assert.Equal(t, 1, s.ConnectionStatistics.Successful)
	assert.Equal(t, 1, s.ConnectionStatistics.Attempted)
}

func interacted(c interactionCounter, times int) func() bool {
	return func() bool {
		return c.interactions() == times
	}
}
