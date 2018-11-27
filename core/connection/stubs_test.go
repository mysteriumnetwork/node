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

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/session"
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

type fakeState string

const (
	processStarted      fakeState = "processStarted"
	connectingState     fakeState = "connectingState"
	reconnectingState   fakeState = "reconnectingState"
	waitState           fakeState = "waitState"
	authenticatingState fakeState = "authenticatingState"
	getConfigState      fakeState = "getConfigState"
	assignIPState       fakeState = "assignIPState"
	connectedState      fakeState = "connectedState"
	exitingState        fakeState = "exitingState"
	processExited       fakeState = "processExited"
)

type connectionFactoryFake struct {
	mockError      error
	mockConnection *connectionFake
}

func (cff *connectionFactoryFake) CreateConnection(connectionParams ConnectOptions, stateChannel StateChannel, statisticsChannel StatisticsChannel) (Connection, error) {
	//each test can set this value to simulate connection creation error, this flag is reset BEFORE each test
	if cff.mockError != nil {
		return nil, cff.mockError
	}

	stateCallback := func(state fakeState) {
		if state == connectedState {
			stateChannel <- Connected
			statisticsChannel <- cff.mockConnection.onStartReportStats
		}
		if state == exitingState {
			stateChannel <- Disconnecting
		}
		if state == reconnectingState {
			stateChannel <- Reconnecting
		}
		//this is the last state - close channel (according to best practices of go - channel writer controls channel)
		if state == processExited {
			close(stateChannel)
		}
	}
	cff.mockConnection.StateCallback(stateCallback)

	// we copy the values over, so that the factory always returns a new instance of connection
	copy := connectionFake{
		onStartReportStates: cff.mockConnection.onStartReportStates,
		onStartReturnError:  cff.mockConnection.onStartReturnError,
		onStopReportStates:  cff.mockConnection.onStopReportStates,
		stateCallback:       cff.mockConnection.stateCallback,
		onStartReportStats:  cff.mockConnection.onStartReportStats,
		fakeProcess:         sync.WaitGroup{},
	}

	return &copy, nil
}

type connectionFake struct {
	onStartReturnError  error
	onStartReportStates []fakeState
	onStopReportStates  []fakeState
	stateCallback       func(state fakeState)
	onStartReportStats  consumer.SessionStatistics
	fakeProcess         sync.WaitGroup
	sync.RWMutex
}

func (foc *connectionFake) Start() error {
	foc.RLock()
	defer foc.RUnlock()

	if foc.onStartReturnError != nil {
		return foc.onStartReturnError
	}

	foc.fakeProcess.Add(1)
	for _, fakeState := range foc.onStartReportStates {
		foc.reportState(fakeState)
	}
	return nil
}

func (foc *connectionFake) Wait() error {
	foc.fakeProcess.Wait()
	return nil
}

func (foc *connectionFake) Stop() {
	for _, fakeState := range foc.onStopReportStates {
		foc.reportState(fakeState)
	}
	foc.fakeProcess.Done()
}

func (foc *connectionFake) reportState(state fakeState) {
	foc.RLock()
	defer foc.RUnlock()

	foc.stateCallback(state)
}

func (foc *connectionFake) StateCallback(callback func(state fakeState)) {
	foc.Lock()
	defer foc.Unlock()

	foc.stateCallback = callback
}

type fakeDialog struct {
	peerID    identity.Identity
	sessionID session.ID

	closed bool
	sync.RWMutex
}

func (fd *fakeDialog) PeerID() identity.Identity {
	fd.RLock()
	defer fd.RUnlock()

	return fd.peerID
}

func (fd *fakeDialog) Close() error {
	fd.Lock()
	defer fd.Unlock()

	fd.closed = true
	return nil
}

func (fd *fakeDialog) Receive(consumer communication.MessageConsumer) error {
	return nil
}
func (fd *fakeDialog) Respond(consumer communication.RequestConsumer) error {
	return nil
}

func (fd *fakeDialog) Send(producer communication.MessageProducer) error {
	return nil
}

func (fd *fakeDialog) Request(producer communication.RequestProducer) (responsePtr interface{}, err error) {
	return &session.CreateResponse{
			Success: true,
			Session: session.SessionDto{
				ID:     fd.sessionID,
				Config: []byte("{}"),
			},
		},
		nil
}
