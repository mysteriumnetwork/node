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

import "sync"

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
	vpnClientCreationError error
	fakeVpnClient          *vpnClientFake
}

func (cff *connectionFactoryFake) CreateConnection(connectionParams ConnectOptions, stateChannel StateChannel) (Connection, error) {
	//each test can set this value to simulate openvpn creation error, this flag is reset BEFORE each test
	if cff.vpnClientCreationError != nil {
		return nil, cff.vpnClientCreationError
	}
	stateCallback := func(state fakeState) {
		if state == connectedState {
			stateChannel <- Connected
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
	cff.fakeVpnClient.StateCallback(stateCallback)
	return cff.fakeVpnClient, nil
}

type vpnClientFake struct {
	onStartReturnError  error
	onStartReportStates []fakeState
	onStopReportStates  []fakeState
	stateCallback       func(state fakeState)
	fakeProcess         sync.WaitGroup

	sync.RWMutex
}

func (foc *vpnClientFake) Start() error {
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

func (foc *vpnClientFake) Wait() error {
	foc.fakeProcess.Wait()
	return nil
}

func (foc *vpnClientFake) Stop() {
	for _, fakeState := range foc.onStopReportStates {
		foc.reportState(fakeState)
	}
	foc.fakeProcess.Done()
}

func (foc *vpnClientFake) reportState(state fakeState) {
	foc.RLock()
	defer foc.RUnlock()

	foc.stateCallback(state)
}

func (foc *vpnClientFake) StateCallback(callback func(state fakeState)) {
	foc.Lock()
	defer foc.Unlock()

	foc.stateCallback = callback
}
