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
	"sync"

	stats_dto "github.com/mysteriumnetwork/node/client/stats/dto"
	"github.com/mysteriumnetwork/node/service_discovery/dto"
)

type fakePromiseIssuer struct {
	startCalled bool
	stopCalled  bool
}

func (issuer *fakePromiseIssuer) Start(proposal dto.ServiceProposal) error {
	issuer.startCalled = true
	return nil
}

func (issuer *fakePromiseIssuer) Stop() error {
	issuer.stopCalled = true
	return nil
}

// StubPublisherEvent represents the event in publishers history
type StubPublisherEvent struct {
	calledWithTopic string
	calledWithArgs  []interface{}
}

// StubPublisher acts as a publisher
type StubPublisher struct {
	publishHistory []StubPublisherEvent
	lock           sync.Mutex
}

// NewStubPublisher returns a stub publisher
func NewStubPublisher() *StubPublisher {
	return &StubPublisher{
		publishHistory: make([]StubPublisherEvent, 0),
	}
}

// Publish adds the given event to the publish history
func (sp *StubPublisher) Publish(topic string, args ...interface{}) {
	sp.lock.Lock()
	defer sp.lock.Unlock()
	event := StubPublisherEvent{
		calledWithTopic: topic,
		calledWithArgs:  args,
	}
	sp.publishHistory = append(sp.publishHistory, event)
}

// Clear clears the event history
func (sp *StubPublisher) Clear() {
	sp.lock.Lock()
	defer sp.lock.Unlock()
	sp.publishHistory = make([]StubPublisherEvent, 0)
}

// GetEventHistory fetches the event history
func (sp *StubPublisher) GetEventHistory() []StubPublisherEvent {
	sp.lock.Lock()
	defer sp.lock.Unlock()
	return sp.publishHistory
}

// StubSessionStorer allows us to get all sessions, save and update them
type StubSessionStorer struct {
	SaveError    error
	SaveCalled   bool
	UpdateError  error
	UpdateCalled bool
	GetAllCalled bool
	GetAllError  error
}

func (sss *StubSessionStorer) Save(object interface{}) error {
	sss.SaveCalled = true
	return sss.SaveError
}

func (sss *StubSessionStorer) Update(object interface{}) error {
	sss.UpdateCalled = true
	return sss.UpdateError
}

func (sss *StubSessionStorer) GetAll(array interface{}) error {
	sss.GetAllCalled = true
	return sss.GetAllError
}

type StubRetriever struct {
	Value stats_dto.SessionStats
}

func (sr *StubRetriever) Retrieve() stats_dto.SessionStats {
	return sr.Value
}
