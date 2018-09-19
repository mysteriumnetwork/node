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

	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
)

var (
	expectedID      = SessionID("mocked-id")
	expectedSession = Session{
		ID:         expectedID,
		Config:     mockedVPNConfig,
		ConsumerID: identity.FromAddress("deadbeef"),
	}
	lastSession Session
)

const mockedVPNConfig = "config_string"

func mockedConfigProvider() (ServiceConfiguration, error) {
	return mockedVPNConfig, nil
}

func saveSession(sessionInstance Session) {
	lastSession = sessionInstance
}

func TestManager_Create(t *testing.T) {
	manager := NewManager(
		func() SessionID {
			return expectedID
		},
		mockedConfigProvider,
		saveSession,
	)

	sessionInstance, err := manager.Create(identity.FromAddress("deadbeef"))
	assert.NoError(t, err)
	assert.Exactly(t, expectedSession, sessionInstance)
	assert.Exactly(t, expectedSession, lastSession)
}
