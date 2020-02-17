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
	"errors"
	"sync"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session"
	"github.com/rs/zerolog/log"
)

// StubPublisherEvent represents the event in publishers history
type StubPublisherEvent struct {
	calledWithTopic string
	calledWithData  interface{}
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
func (sp *StubPublisher) Publish(topic string, data interface{}) {
	sp.lock.Lock()
	defer sp.lock.Unlock()
	event := StubPublisherEvent{
		calledWithTopic: topic,
		calledWithData:  data,
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
	mockConnection *connectionMock
}

func (c *connectionFactoryFake) CreateConnection(serviceType string) (Connection, error) {
	//each test can set this value to simulate connection creation error, this flag is reset BEFORE each test
	if c.mockError != nil {
		return nil, c.mockError
	}

	c.mockConnection.stateChannel = make(chan State, 100)
	c.mockConnection.statisticsChannel = make(chan consumer.SessionStatistics, 100)

	stateCallback := func(state fakeState) {
		if state == connectedState {
			c.mockConnection.stateChannel <- Connected
			c.mockConnection.statisticsChannel <- c.mockConnection.onStartReportStats
		}
		if state == exitingState {
			c.mockConnection.stateChannel <- Disconnecting
		}
		if state == reconnectingState {
			c.mockConnection.stateChannel <- Reconnecting
		}
		//this is the last state - close channel (according to best practices of go - channel writer controls channel)
		if state == processExited {
			close(c.mockConnection.stateChannel)
		}
	}
	c.mockConnection.StateCallback(stateCallback)

	// we copy the values over, so that the factory always returns a new instance of connection
	copy := connectionMock{
		stateChannel:        c.mockConnection.stateChannel,
		statisticsChannel:   c.mockConnection.statisticsChannel,
		onStartReportStates: c.mockConnection.onStartReportStates,
		onStartReturnError:  c.mockConnection.onStartReturnError,
		onStopReportStates:  c.mockConnection.onStopReportStates,
		stateCallback:       c.mockConnection.stateCallback,
		onStartReportStats:  c.mockConnection.onStartReportStats,
		fakeProcess:         sync.WaitGroup{},
		stopBlock:           c.mockConnection.stopBlock,
	}

	return &copy, nil
}

type connectionMock struct {
	stateChannel        chan State
	statisticsChannel   chan consumer.SessionStatistics
	onStartReturnError  error
	onStartReportStates []fakeState
	onStopReportStates  []fakeState
	stateCallback       func(state fakeState)
	onStartReportStats  consumer.SessionStatistics
	fakeProcess         sync.WaitGroup
	stopBlock           chan struct{}
	sync.RWMutex
}

func (foc *connectionMock) State() <-chan State {
	return foc.stateChannel
}

func (foc *connectionMock) Statistics() <-chan consumer.SessionStatistics {
	return foc.statisticsChannel
}

func (foc *connectionMock) GetConfig() (ConsumerConfig, error) {
	return nil, nil
}

func (foc *connectionMock) Start(connectionParams ConnectOptions) error {
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

func (foc *connectionMock) Wait() error {
	foc.fakeProcess.Wait()
	return nil
}

func (foc *connectionMock) Stop() {
	for _, fakeState := range foc.onStopReportStates {
		foc.reportState(fakeState)
	}
	if foc.stopBlock != nil {
		<-foc.stopBlock
	}
	foc.fakeProcess.Done()
}

func (foc *connectionMock) reportState(state fakeState) {
	foc.RLock()
	defer foc.RUnlock()

	foc.stateCallback(state)
}

func (foc *connectionMock) StateCallback(callback func(state fakeState)) {
	foc.Lock()
	defer foc.Unlock()

	foc.stateCallback = callback
}

type mockDialog struct {
	peerID      identity.Identity
	sessionID   session.ID
	paymentInfo session.PaymentInfo
	closed      bool
	sync.RWMutex
}

func (md *mockDialog) PeerID() identity.Identity {
	md.RLock()
	defer md.RUnlock()

	return md.peerID
}

func (md *mockDialog) assertNotClosed() {
	md.RLock()
	defer md.RUnlock()

	if md.closed {
		panic("Incorrect dialog handling! dialog was closed already")
	}
}

func (md *mockDialog) Close() error {
	md.Lock()
	defer md.Unlock()

	md.closed = true
	log.Info().Msg("Dialog closed")
	return nil
}

func (md *mockDialog) Receive(consumer communication.MessageConsumer) error {
	md.assertNotClosed()
	return nil
}
func (md *mockDialog) Respond(consumer communication.RequestConsumer) error {
	md.assertNotClosed()
	return nil
}

func (md *mockDialog) ReceiveUnsubscribe(endpoint communication.MessageEndpoint) {
	md.assertNotClosed()
}

func (md *mockDialog) Unsubscribe() {
	md.assertNotClosed()
}

func (md *mockDialog) Send(producer communication.MessageProducer) error {
	return nil
}

var ErrUnknownRequest = errors.New("unknown request")

func (md *mockDialog) Request(producer communication.RequestProducer) (responsePtr interface{}, err error) {
	md.assertNotClosed()
	if producer.GetRequestEndpoint() == communication.RequestEndpoint("session-destroy") {
		return &session.DestroyResponse{
				Success: true,
			},
			nil
	}

	if producer.GetRequestEndpoint() == communication.RequestEndpoint("session-create") {
		return &session.CreateResponse{
				Success: true,
				Session: session.SessionDto{
					ID:     md.sessionID,
					Config: []byte("{}"),
				},
				PaymentInfo: md.paymentInfo,
			},
			nil
	}
	return nil, ErrUnknownRequest
}
