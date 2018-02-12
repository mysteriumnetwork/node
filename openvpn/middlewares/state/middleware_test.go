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
		{">STATE:1495493709,AUTH,,,,,,", openvpn.STATE_AUTH},
		{">STATE:1495891020,RECONNECTING,ping-restart,,,,,", openvpn.STATE_RECONNECTING},
		{">STATE:1495891025,WAIT,,,,,,", openvpn.STATE_WAIT},
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
			openvpn.STATE_CONNECTING,
			openvpn.STATE_ASSIGN_IP,
			openvpn.STATE_CONNECTED,
			openvpn.STATE_EXITING,
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
