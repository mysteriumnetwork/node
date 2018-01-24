package auth

import (
	"github.com/mysterium/node/openvpn"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

var currentState openvpn.State

type fakeAuthenticator struct {
}

func (a *fakeAuthenticator) auth() (username string, password string, err error) {
	return
}

func (a *fakeAuthenticator) authWithValid() (username string, password string, err error) {
	username = "valid_username"
	password = "valid_password"
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
	authenticator := &fakeAuthenticator{}
	middleware := NewMiddleware(authenticator.auth)
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
	authenticator := &fakeAuthenticator{}
	middleware := NewMiddleware(authenticator.auth)

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

	authenticator := &fakeAuthenticator{}
	middleware := NewMiddleware(authenticator.auth)
	connection := &fakeConnection{}
	middleware.Start(connection)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
	}
}
