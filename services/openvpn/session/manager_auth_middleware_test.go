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

package session

import (
	"testing"

	"github.com/mysteriumnetwork/go-openvpn/openvpn/management"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/server/auth"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session"
	"github.com/stretchr/testify/assert"
)

var (
	identityToExtract = identity.FromAddress("deadbeef")
	validator         = mockValidatorWithSession(identityToExtract, session.Session{
		ID:         session.ID("Boop!"),
		ConsumerID: identityToExtract,
	})
)

type fakeAuthenticatorStub struct {
	username      string
	password      string
	called        bool
	authenticated bool
}

func (f *fakeAuthenticatorStub) fakeAuthenticator(clientID int, username, password string) (bool, error) {
	f.called = true
	f.username = username
	f.password = password
	return f.authenticated, nil
}

func (f *fakeAuthenticatorStub) fakeSessionCleanup(username string) error {
	f.called = true
	f.username = username
	return nil
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

	return validator.Validate(clientID, username, password)
}

func TestMiddlewareConsumesClientIdsAntKeysWithSeveralDigits(t *testing.T) {
	var tests = []string{
		">CLIENT:CONNECT,115,23",
		">CLIENT:REAUTH,11,27",
	}

	fas := newFakeAuthenticatorStub()
	middleware := auth.NewMiddleware(fas.fakeAuthenticator)
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

	middleware := auth.NewMiddleware(fas.fakeAuthenticator)
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

func TestSecondClientWithTheSameCredentialsIsConnected(t *testing.T) {
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
	middleware := auth.NewMiddleware(fas.newFakeSessionValidator)

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
	// second authentication with the same credentials but with different clientID should succeed

	assert.Equal(t, "client-auth-nt 2 4", mockMangement.LastLine)
}

func feedLinesToMiddleware(middleware management.Middleware, lines []string) {
	for _, line := range lines {
		middleware.ConsumeLine(line)
	}
}
