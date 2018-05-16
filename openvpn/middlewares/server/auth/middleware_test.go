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

package auth

import (
	"github.com/mysterium/node/openvpn/management"
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/mysterium/node/session"
	ovpnsession "github.com/mysterium/node/openvpn/session"
	"github.com/mysterium/node/identity"
)

type fakeAuthenticatorStub struct {
	username      string
	password      string
	called        bool
	authenticated bool
	manager 	  session.Manager
}

func (f *fakeAuthenticatorStub) fakeAuthenticator(clientID int, username, password string) (bool, error) {
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

func (f *fakeAuthenticatorStub) newFakeSessionValidator(clientID int, username, password string) (bool, error) {
	f.called = true
	f.username = username
	f.password = password

	// create session before validating its exists
	f.manager.Create(identity.FromAddress("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68"))

	// log.Info("authenticating user: ", username, " password: ", password)

	sessionValidator := ovpnsession.NewSessionValidatorWithClientID(
		f.manager.FindUpdateSessionWithClientID,
		identity.NewExtractor(),
	)

	f.authenticated = true

	return sessionValidator(clientID, username, password)
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

var mockedVPNConfig = "config_string"

type mockedConfigProvider func() string

func (mcp mockedConfigProvider) ProvideServiceConfig() (session.ServiceConfiguration, error) {
	return mcp(), nil
}

func provideMockedVPNConfig() string {
	return mockedVPNConfig
}

func TestSecondClientWithTheSameCredentialsIsDisconnected(t *testing.T) {
	var firstClientConnected = []string{
		">CLIENT:CONNECT,1,4",
		">CLIENT:ENV,username=Boop!",
		">CLIENT:ENV,password=V6ifmvLuAT+hbtLBX/0xm3C0afywxTIdw1HqLmA4onpwmibHbxVhl50Gr3aRUZMqw1WxkfSIVdhpbCluHGBKsgE=",
		">CLIENT:ENV,END",
	}

	var secondClientDisconnected = []string{
		">CLIENT:CONNECT,2,4",
		">CLIENT:ENV,username=Boop!",
		">CLIENT:ENV,password=V6ifmvLuAT+hbtLBX/0xm3C0afywxTIdw1HqLmA4onpwmibHbxVhl50Gr3aRUZMqw1WxkfSIVdhpbCluHGBKsgE=",
		">CLIENT:ENV,END",
	}

	fas := newFakeAuthenticatorStub()
	fas.manager = ovpnsession.NewManager(
		mockedConfigProvider(provideMockedVPNConfig),
		&session.GeneratorFake{
			SessionIdMock: session.SessionID("Boop!"),
		},
	)

	middleware := NewMiddleware(fas.newFakeSessionValidator)

	mockMangement := &management.MockConnection{
		CommandResult: "SUCCESS",
	}
	middleware.Start(mockMangement)

	feedLinesToMiddleware(middleware, firstClientConnected)

	assert.True(t, fas.called)
	assert.Equal(t, "Boop!", fas.username)
	assert.Equal(t, "V6ifmvLuAT+hbtLBX/0xm3C0afywxTIdw1HqLmA4onpwmibHbxVhl50Gr3aRUZMqw1WxkfSIVdhpbCluHGBKsgE=", fas.password)
	assert.Equal(t, "client-auth-nt 1 4", mockMangement.LastLine)

	fas.reset()
	feedLinesToMiddleware(middleware, secondClientDisconnected)
	assert.True(t, fas.called)
	assert.Equal(t, "Boop!", fas.username)
	assert.Equal(t, "V6ifmvLuAT+hbtLBX/0xm3C0afywxTIdw1HqLmA4onpwmibHbxVhl50Gr3aRUZMqw1WxkfSIVdhpbCluHGBKsgE=", fas.password)
	// second authentication with the same credentials but with different clientID should fail
	assert.Equal(t, "client-deny 2 4 wrong username or password", mockMangement.LastLine)
}

func feedLinesToMiddleware(middleware management.Middleware, lines []string) {
	for _, line := range lines {
		middleware.ConsumeLine(line)
	}
}
