package state

import (
	"github.com/mysterium/node/openvpn"
	"github.com/stretchr/testify/assert"
	"testing"
)

var currentState openvpn.State

func updateCurrentState(state openvpn.State) error {
	currentState = state
	return nil
}

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
	middleware.Subscribe(updateCurrentState)
	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
		assert.Equal(t, test.expectedState, currentState, test.line)
	}
}
