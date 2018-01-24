package auth

import (
	"github.com/mysterium/node/openvpn"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

var currentState openvpn.State

type FakeAuthenticatorChecker struct {
	LastUsername  string
	LastPassword  string
	authenticator func(username, password string) (bool, error)
}

func (fac *FakeAuthenticatorChecker) NewFakeAuthenticatorChecker() FakeAuthenticatorChecker {
	return FakeAuthenticatorChecker{
		LastUsername: "",
		LastPassword: "",
		authenticator: func(username, password string) (bool, error) {
			fac.LastUsername = username
			fac.LastPassword = password
			if username == "bad" {
				return false, nil
			}

			return true, nil
		},
	}
}

func NewFakeMiddleware(authenticator *FakeAuthenticatorChecker) openvpn.ManagementMiddleware {
	return &middleware{
		authenticator: authenticator.authenticator,
		connection:    nil,
	}
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
	var fac FakeAuthenticatorChecker
	authenticator := fac.NewFakeAuthenticatorChecker()
	middleware := NewMiddleware(authenticator.authenticator)
	assert.NotNil(t, middleware)
}

func Test_ConsumeLineSkips(t *testing.T) {
	var tests = []struct {
		line string
	}{
		{">SOME_LINE_TO_BE_DELIVERED"},
		{">ANOTHER_LINE_TO_BE_DELIVERED"},
	}
	var fac FakeAuthenticatorChecker
	authenticator := fac.NewFakeAuthenticatorChecker()
	middleware := NewMiddleware(authenticator.authenticator)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.False(t, consumed, test.line)
	}
}

func Test_ConsumeLineTakes(t *testing.T) {
	var tests = []struct {
		line string
	}{
		{">CLIENT:REAUTH,0,0"},
		{">CLIENT:CONNECT,0,0"},
		{">CLIENT:ENV,password=12341234"},
		{">CLIENT:ENV,username=username"},
	}

	var fac FakeAuthenticatorChecker
	authenticator := fac.NewFakeAuthenticatorChecker()
	middleware := NewMiddleware(authenticator.authenticator)
	connection := &fakeConnection{}
	middleware.Start(connection)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
	}
}

func Test_ConsumeLineAuthState(t *testing.T) {
	var tests = []struct {
		line          string
		expectedState openvpn.State
	}{
		{">CLIENT:REAUTH,0,0", openvpn.STATE_AUTH},
		{">CLIENT:CONNECT,0,0", openvpn.STATE_AUTH},
	}

	for _, test := range tests {
		var fac FakeAuthenticatorChecker
		authenticator := fac.NewFakeAuthenticatorChecker()
		middleware := NewMiddleware(authenticator.authenticator)
		connection := &fakeConnection{}
		middleware.Start(connection)

		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
		assert.Equal(t, test.expectedState, middleware.State())
	}
}

func Test_ConsumeLineNotAuthState(t *testing.T) {
	var tests = []struct {
		line            string
		unexpectedState openvpn.State
	}{
		{">CLIENT:ENV,password=12341234", openvpn.STATE_AUTH},
		{">CLIENT:ENV,username=username", openvpn.STATE_AUTH},
	}

	for _, test := range tests {
		var fac FakeAuthenticatorChecker
		authenticator := fac.NewFakeAuthenticatorChecker()
		middleware := NewMiddleware(authenticator.authenticator)
		connection := &fakeConnection{}
		middleware.Start(connection)

		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.False(t, consumed, test.line)
		assert.NotEqual(t, test.unexpectedState, middleware.State())
	}
}

func Test_ConsumeLineParseChecker(t *testing.T) {
	var tests = []struct {
		line          string
		expectedState openvpn.State
	}{
		{">CLIENT:CONNECT,0,0", openvpn.STATE_AUTH},
		{">CLIENT:ENV,password=12341234", openvpn.STATE_AUTH},
		{">CLIENT:ENV,username=username1", openvpn.STATE_AUTH},
		{">CLIENT:ENV,END", openvpn.STATE_UNDEFINED},
	}
	var fac FakeAuthenticatorChecker
	authenticator := fac.NewFakeAuthenticatorChecker()

	middleware := NewMiddleware(authenticator.authenticator)
	connection := &fakeConnection{}
	middleware.Start(connection)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
		assert.Equal(t, test.expectedState, middleware.State())
	}
	assert.Equal(t, fac.LastUsername, "username1")
	assert.Equal(t, fac.LastPassword, "12341234")
}

func Test_ConsumeLineAuthTrueChecker(t *testing.T) {
	var tests = []struct {
		line          string
		expectedState openvpn.State
	}{
		{">CLIENT:CONNECT,0,0", openvpn.STATE_AUTH},
		{">CLIENT:ENV,password=12341234", openvpn.STATE_AUTH},
		{">CLIENT:ENV,username=username1", openvpn.STATE_AUTH},
		{">CLIENT:ENV,END", openvpn.STATE_UNDEFINED},
	}
	var fac FakeAuthenticatorChecker
	authenticator := fac.NewFakeAuthenticatorChecker()
	authFake := NewAuthenticatorFake()
	middleware := NewMiddleware(authenticator.authenticator)
	connection := &fakeConnection{}
	middleware.Start(connection)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
		assert.Equal(t, test.expectedState, middleware.State())
	}
	authenticated, _ := authFake(fac.LastUsername, fac.LastPassword)
	assert.True(t, authenticated)
}

func Test_ConsumeLineAuthFalseChecker(t *testing.T) {
	var tests = []struct {
		line          string
		expectedState openvpn.State
	}{
		{">CLIENT:CONNECT,0,0", openvpn.STATE_AUTH},
		{">CLIENT:ENV,username=bad", openvpn.STATE_AUTH},
		{">CLIENT:ENV,password=12341234", openvpn.STATE_AUTH},
		{">CLIENT:ENV,END", openvpn.STATE_UNDEFINED},
	}
	var fac FakeAuthenticatorChecker
	authenticator := fac.NewFakeAuthenticatorChecker()
	authFake := NewAuthenticatorFake()
	middleware := NewMiddleware(authenticator.authenticator)
	connection := &fakeConnection{}
	middleware.Start(connection)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
		assert.Equal(t, test.expectedState, middleware.State())
	}
	authenticated, _ := authFake(fac.LastUsername, fac.LastPassword)
	assert.False(t, authenticated)
}
