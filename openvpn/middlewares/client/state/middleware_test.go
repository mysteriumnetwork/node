package state

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var currentState State

func updateCurrentState(state State) error {
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
		expectedState State
	}{
		{">STATE:1495493709,AUTH,,,,,,", STATE_AUTH},
		{">STATE:1495891020,RECONNECTING,ping-restart,,,,,", STATE_RECONNECTING},
		{">STATE:1495891025,WAIT,,,,,,", STATE_WAIT},
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
