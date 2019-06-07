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
	"github.com/mysteriumnetwork/node/nat"
	natEvent "github.com/mysteriumnetwork/node/nat/event"
	"github.com/mysteriumnetwork/node/session"
	sessionEvent "github.com/mysteriumnetwork/node/session/event"
	"github.com/stretchr/testify/assert"
)

type debounceTester struct {
	timesCalled int
	lock        sync.Mutex
}

func (dt *debounceTester) do(interface{}) {
	dt.lock.Lock()
	dt.timesCalled++
	dt.lock.Unlock()
}

func (dt *debounceTester) get() int {
	dt.lock.Lock()
	defer dt.lock.Unlock()
	return dt.timesCalled
}

func Test_Debounce_CallsOnceInInterval(t *testing.T) {
	dt := &debounceTester{}
	duration := time.Microsecond * 1000
	f := debounce(dt.do, duration)
	for i := 1; i < 10; i++ {
		f(struct{}{})
	}

	select {
	case <-time.After(duration * 5):
		assert.Equal(t, 1, dt.get())
	}
}

var mockNATStatus = nat.Status{
	Status: "status",
	Error:  errors.New("err"),
}

type natStatusProviderMock struct {
	statusToReturn nat.Status
	timesCalled    int
	lock           sync.Mutex
}

func (nspm *natStatusProviderMock) Status() nat.Status {
	nspm.lock.Lock()
	defer nspm.lock.Unlock()
	nspm.timesCalled++
	return nspm.statusToReturn
}

func (nspm *natStatusProviderMock) getTimesCalled() int {
	nspm.lock.Lock()
	defer nspm.lock.Unlock()
	return nspm.timesCalled
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

func (mp *mockPublisher) getLast() (string, interface{}) {
	mp.lock.Lock()
	defer mp.lock.Unlock()
	return mp.publishedTopic, mp.publishedData
}

type serviceListerMock struct {
	lock             sync.Mutex
	timesCalled      int
	servicesToReturn map[service.ID]*service.Instance
}

func (slm *serviceListerMock) getTimesCalled() int {
	slm.lock.Lock()
	defer slm.lock.Unlock()
	return slm.timesCalled
}

func (slm *serviceListerMock) List() map[service.ID]*service.Instance {
	slm.lock.Lock()
	defer slm.lock.Unlock()
	slm.timesCalled++
	return slm.servicesToReturn
}

type serviceSessionStorageMock struct {
	timesCalled      int
	sessionsToReturn []session.Session
	lock             sync.Mutex
}

func (sssm *serviceSessionStorageMock) GetAll() []session.Session {
	sssm.lock.Lock()
	defer sssm.lock.Unlock()
	sssm.timesCalled++
	return sssm.sessionsToReturn
}

func (sssm *serviceSessionStorageMock) getTimesCalled() int {
	sssm.lock.Lock()
	defer sssm.lock.Unlock()
	return sssm.timesCalled
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

	select {
	case <-time.After(duration * 3):
		assert.Equal(t, 1, natProvider.getTimesCalled())
	}

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
		keeper.ConsumeSessionEvent(sessionEvent.Payload{})
	}

	select {
	case <-time.After(duration * 3):
		assert.Equal(t, 1, sessionStorage.getTimesCalled())
	}
	assert.Equal(t, string(expected.ID), keeper.GetState().Sessions[0].ID)
	assert.Equal(t, expected.ConsumerID.Address, keeper.GetState().Sessions[0].ConsumerID)
	assert.True(t, expected.CreatedAt.Equal(keeper.GetState().Sessions[0].CreatedAt))
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

	select {
	case <-time.After(duration * 3):
		assert.Equal(t, 1, sl.getTimesCalled())
	}

	actual := keeper.GetState().Services[0]
	assert.Equal(t, string(id), actual.ID)
	assert.Equal(t, expected.Proposal().ServiceType, actual.Type)
	assert.Equal(t, expected.Proposal().ProviderID, actual.ProviderID)
	assert.Equal(t, expected.Options(), actual.Options)
	assert.Equal(t, string(expected.State()), actual.Status)
	assert.EqualValues(t, expected.Proposal(), actual.Proposal)
}
