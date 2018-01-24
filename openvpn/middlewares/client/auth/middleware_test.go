package auth

import (
	"github.com/mysterium/node/openvpn"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func auth() (username string, password string, err error) {
	return
}

type fakeConnection struct {
	lastDataWritten []byte
	net.Conn
}

func (conn *fakeConnection) Read(b []byte) (int, error) {
	return 0, nil
}

func (conn *fakeConnection) Write(b []byte) (n int, err error) {
	conn.lastDataWritten = b
	return 0, nil
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
	var tests = []struct {
		line          string
		expectedState openvpn.State
	}{
		{">PASSWORD:Need 'Auth' username/password", openvpn.STATE_AUTH},
	}

	middleware := NewMiddleware(auth)
	connection := &fakeConnection{}
	middleware.Start(connection)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
	}
}
