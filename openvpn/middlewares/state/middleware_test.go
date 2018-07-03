/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/management"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Factory(t *testing.T) {
	middleware := NewMiddleware()
	assert.NotNil(t, middleware)
}

func Test_ConsumeLineSkips(t *testing.T) {
	var tests = []struct {
		line string
	}{
		{"OTHER"},
		{"STATE"},
	}

	middleware := NewMiddleware()
	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.False(t, consumed, test.line)
	}
}

func Test_ConsumeLineTakes(t *testing.T) {
	var tests = []struct {
		line          string
		expectedState openvpn.State
	}{
		{">STATE:1495493709,AUTH,,,,,,", openvpn.AuthenticatingState},
		{">STATE:1495891020,RECONNECTING,ping-restart,,,,,", openvpn.ReconnectingState},
		{">STATE:1495891025,WAIT,,,,,,", openvpn.WaitState},
	}

	middleware := &middleware{}
	stateTracker := &stateTracker{}
	middleware.Subscribe(stateTracker.recordState)
	for _, test := range tests {
		stateTracker.reset()
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
		assert.Equal(t, test.expectedState, stateTracker.states[0], test.line)
	}
}

func Test_StartCommandWritesExpectedStringToConnection(t *testing.T) {
	middleware := &middleware{}
	stateTracker := &stateTracker{}
	middleware.Subscribe(stateTracker.recordState)

	mockConnection := &management.MockConnection{}
	mockConnection.CommandResult = "Success!"
	mockConnection.MultilineResponse = []string{
		"1495493709,CONNECTING,,,,,,",
		"1518445456,ASSIGN_IP,,10.8.0.1,,,,",
		"1495493709,CONNECTED,,,,,,",
		"1495493709,EXITING,,,,,,",
	}
	err := middleware.Start(mockConnection)
	assert.NoError(t, err)
	assert.Equal(t, "state on all", mockConnection.LastLine)
	assert.Equal(t,
		[]openvpn.State{
			openvpn.ProcessStarted,
			openvpn.ConnectingState,
			openvpn.AssignIpState,
			openvpn.ConnectedState,
			openvpn.ExitingState,
		},
		stateTracker.states,
	)
}

type stateTracker struct {
	states []openvpn.State
}

func (st *stateTracker) recordState(state openvpn.State) {
	st.states = append(st.states, state)
}

func (st *stateTracker) reset() {
	st.states = nil
}
