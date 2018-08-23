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

	"github.com/mysterium/node/identity"
	"github.com/stretchr/testify/assert"
)

const mockedVPNConfig = "config_string"

var expectedSession = Session{
	ID:         SessionID("mocked-id"),
	Config:     mockedVPNConfig,
	ConsumerID: identity.FromAddress("deadbeef"),
}

type mockedConfigProvider func() string

func (mcp mockedConfigProvider) ProvideServiceConfig() (ServiceConfiguration, error) {
	return mcp(), nil
}

func provideMockedVPNConfig() string {
	return mockedVPNConfig
}

func TestManagerCreatesNewSession(t *testing.T) {
	manager := NewManager(
		mockedConfigProvider(provideMockedVPNConfig),
		&GeneratorFake{
			SessionIdMock: SessionID("mocked-id"),
		},
	)

	sessionInstance, err := manager.Create(identity.FromAddress("deadbeef"))
	assert.NoError(t, err)
	assert.Exactly(t, expectedSession, sessionInstance)

	expectedSessionMap := make(map[SessionID]Session)
	expectedSessionMap[expectedSession.ID] = expectedSession
	assert.Exactly(
		t,
		expectedSessionMap,
		manager.sessionMap,
	)
}

func TestManagerLookupsExistingSession(t *testing.T) {
	sessionMap := make(map[SessionID]Session)
	sessionMap[expectedSession.ID] = expectedSession

	manager := manager{
		sessionMap: sessionMap,
	}

	session, found := manager.FindSession(SessionID("mocked-id"))
	assert.True(t, found)
	assert.Exactly(t, expectedSession, session)
}
