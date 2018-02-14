package auth

import (
	"github.com/mysterium/node/openvpn"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

type fakeAuthenticatorStub struct {
	called        bool
	authenticated bool
}

func (f *fakeAuthenticatorStub) fakeAuthenticator(username, password string) (bool, error) {
	f.called = true

	if username == "bad" {
		return false, nil
	}

	f.authenticated = true
	return true, nil
}

func newFakeAuthenticatorStub() fakeAuthenticatorStub {
	return fakeAuthenticatorStub{}
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

	fas := newFakeAuthenticatorStub()
	middleware := NewMiddleware(fas.fakeAuthenticator)
	assert.NotNil(t, middleware)
}

func Test_ConsumeLineSkips(t *testing.T) {
	var tests = []struct {
		line string
	}{
		{">SOME_LINE_TO_BE_DELIVERED"},
		{">ANOTHER_LINE_TO_BE_DELIVERED"},
		{">PASSWORD"},
		{">USERNAME"},
	}
	fas := newFakeAuthenticatorStub()
	middleware := NewMiddleware(fas.fakeAuthenticator)

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

	fas := newFakeAuthenticatorStub()
	middleware := NewMiddleware(fas.fakeAuthenticator)
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
		fas := newFakeAuthenticatorStub()
		middleware := NewMiddleware(fas.fakeAuthenticator)
		connection := &fakeConnection{}
		middleware.Start(connection)

		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
		assert.Equal(t, test.expectedState, middleware.state)
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
		fas := newFakeAuthenticatorStub()
		middleware := NewMiddleware(fas.fakeAuthenticator)
		connection := &fakeConnection{}
		middleware.Start(connection)

		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.False(t, consumed, test.line)
		assert.NotEqual(t, test.unexpectedState, middleware.state)
		assert.False(t, fas.called)
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

	fas := newFakeAuthenticatorStub()
	middleware := NewMiddleware(fas.fakeAuthenticator)
	connection := &fakeConnection{}
	middleware.Start(connection)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
		assert.Equal(t, test.expectedState, middleware.state)
	}
	assert.True(t, fas.called)
	assert.Equal(t, "username1", middleware.lastUsername)
	assert.Equal(t, "12341234", middleware.lastPassword)
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
	fas := newFakeAuthenticatorStub()
	middleware := NewMiddleware(fas.fakeAuthenticator)
	connection := &fakeConnection{}
	middleware.Start(connection)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
		assert.Equal(t, test.expectedState, middleware.state)
	}
	assert.True(t, fas.called)
	assert.True(t, fas.authenticated)
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
	fas := newFakeAuthenticatorStub()
	middleware := NewMiddleware(fas.fakeAuthenticator)
	connection := &fakeConnection{}
	middleware.Start(connection)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
		assert.Equal(t, test.expectedState, middleware.state)
	}
	assert.True(t, fas.called)
	assert.False(t, fas.authenticated)
}

func TestMiddlewareConsumesClientIdsAntKeysWithSeveralDigits(t *testing.T) {
	var tests = []string{
		">CLIENT:CONNECT,115,23",
		">CLIENT:REAUTH,11,27",
	}

	fas := newFakeAuthenticatorStub()
	middleware := NewMiddleware(fas.fakeAuthenticator)
	for _, testLine := range tests {
		consumed, err := middleware.ConsumeLine(testLine)
		assert.NoError(t, err, testLine)
		assert.Equal(t, true, consumed, testLine)
	}
}
