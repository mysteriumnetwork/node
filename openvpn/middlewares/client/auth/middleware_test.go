package auth

import (
	"github.com/mysterium/node/openvpn/middlewares"
	"github.com/stretchr/testify/assert"
	"testing"
)

func auth() (string, string, error) {
	return "testuser", "testpassword", nil
}

func Test_Factory(t *testing.T) {
	middleware := NewMiddleware(auth)
	assert.NotNil(t, middleware)
}

func Test_ConsumeLineSkips(t *testing.T) {
	var tests = []struct {
		line string
	}{
		{">SOME_LINE_DELIVERED"},
		{">ANOTHER_LINE_DELIVERED"},
		{">PASSWORD"},
	}
	middleware := NewMiddleware(auth)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.False(t, consumed, test.line)
	}
}

func Test_ConsumeLineTakes(t *testing.T) {
	passwordRequest := ">PASSWORD:Need 'Auth' username/password"

	middleware := NewMiddleware(auth)
	mockCmdWriter := &middlewares.MockCommandWriter{}
	middleware.Start(mockCmdWriter)

	consumed, err := middleware.ConsumeLine(passwordRequest)
	assert.NoError(t, err)
	assert.True(t, consumed)
	assert.Equal(t,
		mockCmdWriter.WrittenLines,
		[]string{
			"password 'Auth' testpassword",
			"username 'Auth' testuser",
		},
	)
}
