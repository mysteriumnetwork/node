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
	"testing"

	"github.com/mysteriumnetwork/go-openvpn/openvpn/management"
	"github.com/stretchr/testify/assert"
)

type fakeAuthenticatorStub struct {
	called        bool
	username      string
	password      string
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

func Test_Factory(t *testing.T) {
	fas := fakeAuthenticatorStub{}
	middleware := NewMiddleware(fas.fakeAuthenticator, fas.fakeSessionCleanup)
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
	fas := fakeAuthenticatorStub{}
	middleware := NewMiddleware(fas.fakeAuthenticator, fas.fakeSessionCleanup)

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

	fas := fakeAuthenticatorStub{}
	middleware := NewMiddleware(fas.fakeAuthenticator, fas.fakeSessionCleanup)
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
		fas := fakeAuthenticatorStub{}
		middleware := NewMiddleware(fas.fakeAuthenticator, fas.fakeSessionCleanup)
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
		fas := fakeAuthenticatorStub{}
		middleware := NewMiddleware(fas.fakeAuthenticator, fas.fakeSessionCleanup)
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
	fas := fakeAuthenticatorStub{}
	fas.authenticated = true
	middleware := NewMiddleware(fas.fakeAuthenticator, fas.fakeSessionCleanup)
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
	fas := fakeAuthenticatorStub{}
	fas.authenticated = false
	middleware := NewMiddleware(fas.fakeAuthenticator, fas.fakeSessionCleanup)
	mockConnection := &management.MockConnection{}
	middleware.Start(mockConnection)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
	}
	assert.Equal(t, "client-deny 3 4 wrong username or password", mockConnection.LastLine)
}
