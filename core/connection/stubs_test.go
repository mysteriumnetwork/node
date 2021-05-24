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
	"context"
	"sync"

	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
)

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
	// each test can set this value to simulate connection creation error, this flag is reset BEFORE each test
	if c.mockError != nil {
		return nil, c.mockError
	}

	c.mockConnection.stateChannel = make(chan connectionstate.State, 100)

	stateCallback := func(state fakeState) {
		if state == connectedState {
			c.mockConnection.stateChannel <- connectionstate.Connected
		}
		if state == exitingState {
			c.mockConnection.stateChannel <- connectionstate.Disconnecting
		}
		if state == reconnectingState {
			c.mockConnection.stateChannel <- connectionstate.Reconnecting
		}
		// this is the last state - close channel (according to best practices of go - channel writer controls channel)
		if state == processExited {
			close(c.mockConnection.stateChannel)
		}
	}
	c.mockConnection.StateCallback(stateCallback)

	// we copy the values over, so that the factory always returns a new instance of connection
	copy := connectionMock{
		stateChannel:        c.mockConnection.stateChannel,
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
	stateChannel        chan connectionstate.State
	onStartReturnError  error
	onStartReportStates []fakeState
	onStopReportStates  []fakeState
	stateCallback       func(state fakeState)
	onStartReportStats  connectionstate.Statistics
	fakeProcess         sync.WaitGroup
	stopBlock           chan struct{}
	sync.RWMutex
}

func (c *connectionMock) State() <-chan connectionstate.State {
	return c.stateChannel
}

func (c *connectionMock) Statistics() (connectionstate.Statistics, error) {
	return c.onStartReportStats, nil
}

func (c *connectionMock) GetConfig() (ConsumerConfig, error) {
	return nil, nil
}

func (c *connectionMock) Reconnect(ctx context.Context, connectionParams ConnectOptions) error {
	return c.Start(ctx, connectionParams)
}

func (c *connectionMock) Start(ctx context.Context, connectionParams ConnectOptions) error {
	c.RLock()
	defer c.RUnlock()

	if c.onStartReturnError != nil {
		return c.onStartReturnError
	}

	c.fakeProcess.Add(1)
	for _, fakeState := range c.onStartReportStates {
		c.reportState(fakeState)
	}
	return nil
}

func (c *connectionMock) Stop() {
	for _, fakeState := range c.onStopReportStates {
		c.reportState(fakeState)
	}
	if c.stopBlock != nil {
		<-c.stopBlock
	}
	c.fakeProcess.Done()
}

func (c *connectionMock) reportState(state fakeState) {
	c.RLock()
	defer c.RUnlock()

	c.stateCallback(state)
}

func (c *connectionMock) StateCallback(callback func(state fakeState)) {
	c.Lock()
	defer c.Unlock()

	c.stateCallback = callback
}
