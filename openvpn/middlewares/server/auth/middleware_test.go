package auth

import (
	"github.com/mysterium/node/openvpn/management"
	"github.com/stretchr/testify/assert"
	"testing"
)

type fakeAuthenticatorStub struct {
	username      string
	password      string
	called        bool
	authenticated bool
}

func (f *fakeAuthenticatorStub) fakeAuthenticator(username, password string) (bool, error) {
	f.called = true
	f.username = username
	f.password = password
	return f.authenticated, nil
}

func (f *fakeAuthenticatorStub) reset() {
	f.called = false
	f.username = ""
	f.password = ""
}

func newFakeAuthenticatorStub() fakeAuthenticatorStub {
	return fakeAuthenticatorStub{}
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
	mockConnection := &management.MockConnection{}
	middleware.Start(mockConnection)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
	}
}

func Test_ConsumeLineAuthState(t *testing.T) {
	var tests = []struct {
		line string
	}{
		{">CLIENT:REAUTH,0,0"},
		{">CLIENT:CONNECT,0,0"},
	}

	for _, test := range tests {
		fas := newFakeAuthenticatorStub()
		middleware := NewMiddleware(fas.fakeAuthenticator)
		mockConnection := &management.MockConnection{}
		middleware.Start(mockConnection)

		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
	}
}

func Test_ConsumeLineNotAuthState(t *testing.T) {
	var tests = []struct {
		line string
	}{
		{">CLIENT:ENV,password=12341234"},
		{">CLIENT:ENV,username=username"},
	}

	for _, test := range tests {
		fas := newFakeAuthenticatorStub()
		middleware := NewMiddleware(fas.fakeAuthenticator)
		mockConnection := &management.MockConnection{}
		middleware.Start(mockConnection)

		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
		assert.False(t, fas.called)
	}
}

func Test_ConsumeLineAuthTrueChecker(t *testing.T) {
	var tests = []struct {
		line string
	}{
		{">CLIENT:CONNECT,1,2"},
		{">CLIENT:ENV,password=12341234"},
		{">CLIENT:ENV,username=username1"},
		{">CLIENT:ENV,END"},
	}
	fas := newFakeAuthenticatorStub()
	fas.authenticated = true
	middleware := NewMiddleware(fas.fakeAuthenticator)
	mockConnection := &management.MockConnection{}
	middleware.Start(mockConnection)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
	}
	assert.True(t, fas.called)
	assert.Equal(t, "username1", fas.username)
	assert.Equal(t, "12341234", fas.password)
	assert.Equal(t, "client-auth-nt 1 2", mockConnection.LastLine)
}

func Test_ConsumeLineAuthFalseChecker(t *testing.T) {
	var tests = []struct {
		line string
	}{
		{">CLIENT:CONNECT,3,4"},
		{">CLIENT:ENV,username=bad"},
		{">CLIENT:ENV,password=12341234"},
		{">CLIENT:ENV,END"},
	}
	fas := newFakeAuthenticatorStub()
	fas.authenticated = false
	middleware := NewMiddleware(fas.fakeAuthenticator)
	mockConnection := &management.MockConnection{}
	middleware.Start(mockConnection)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
	}
	assert.Equal(t, "client-deny 3 4 wrong username or password", mockConnection.LastLine)
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

func TestSecondClientIsNotDisconnectedWhenFirstClientDisconnects(t *testing.T) {
	var firstClientConnected = []string{
		">CLIENT:CONNECT,1,4",
		">CLIENT:ENV,username=client1",
		">CLIENT:ENV,password=passwd1",
		">CLIENT:ENV,END",
	}

	var secondClientConnected = []string{
		">CLIENT:CONNECT,2,4",
		">CLIENT:ENV,username=client2",
		">CLIENT:ENV,password=passwd2",
		">CLIENT:ENV,END",
	}

	var firstClientDisconnected = []string{
		">CLIENT:DISCONNECT,1,4",
		">CLIENT:ENV,username=client1",
		">CLIENT:ENV,password=passwd1",
		">CLIENT:ENV,END",
	}

	fas := newFakeAuthenticatorStub()
	fas.authenticated = true

	mockMangement := &management.MockConnection{
		CommandResult: "SUCCESS",
	}

	middleware := NewMiddleware(fas.fakeAuthenticator)
	middleware.Start(mockMangement)

	feedLinesToMiddleware(middleware, firstClientConnected)

	assert.True(t, fas.called)
	assert.Equal(t, "client1", fas.username)
	assert.Equal(t, "passwd1", fas.password)
	assert.Equal(t, "client-auth-nt 1 4", mockMangement.LastLine)

	fas.reset()
	feedLinesToMiddleware(middleware, secondClientConnected)
	assert.True(t, fas.called)
	assert.Equal(t, "client2", fas.username)
	assert.Equal(t, "passwd2", fas.password)
	assert.Equal(t, "client-auth-nt 2 4", mockMangement.LastLine)

	fas.reset()
	mockMangement.LastLine = ""
	feedLinesToMiddleware(middleware, firstClientDisconnected)
	assert.Empty(t, fas.username)
	assert.Empty(t, fas.password)
	assert.False(t, fas.called)
	assert.Empty(t, mockMangement.LastLine)

}

func feedLinesToMiddleware(middleware management.Middleware, lines []string) {
	for _, line := range lines {
		middleware.ConsumeLine(line)
	}
}
