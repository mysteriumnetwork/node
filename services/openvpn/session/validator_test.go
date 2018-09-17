/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
	"sync"
	"testing"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session"
	"github.com/stretchr/testify/assert"
)

var mockManager = &MockSessionManager{
	session.Session{
		ID:         session.SessionID("fake-id"),
		Config:     mockedVPNConfig,
		ConsumerID: identity.FromAddress("deadbeef"),
	},
	true,
}

var mockExtractor = &MockIdentityExtractor{
	identity.FromAddress("deadbeef"),
	nil,
}

var fakeManager = NewClientMap(mockManager)

var mockValidator = NewValidator(fakeManager, mockExtractor)

func TestValidateReturnsFalseWhenNoSessionFound(t *testing.T) {
	mockExtractor := &MockIdentityExtractor{}

	sessionManager := session.NewManager(
		mockedConfigProvider,
		&session.GeneratorFake{
			SessionIdMock: session.SessionID("mocked-id"),
		},
	)

	manager := &clientMap{sessionManager, make(map[session.SessionID]int), sync.Mutex{}}
	mockValidator := &Validator{manager, mockExtractor}
	authenticated, err := mockValidator.Validate(1, "not important", "not important")

	assert.Errorf(t, err, "no underlying session exists, possible break-in attempt")
	assert.False(t, authenticated)
}

func TestValidateReturnsFalseWhenSignatureIsInvalid(t *testing.T) {
	mockExtractor := &MockIdentityExtractor{
		identity.FromAddress("wrongsignature"),
		nil,
	}

	mockValidator := &Validator{fakeManager, mockExtractor}

	authenticated, err := mockValidator.Validate(1, "not important", "not important")

	assert.NoError(t, err)
	assert.False(t, authenticated)
}

func TestValidateReturnsTrueWhenSessionExistsAndSignatureIsValid(t *testing.T) {
	authenticated, err := mockValidator.Validate(1, "not important", "not important")

	assert.NoError(t, err)
	assert.True(t, authenticated)
}

func TestValidateReturnsFalseWhenSessionExistsAndSignatureIsValidAndClientIDDiffers(t *testing.T) {
	mockValidator.Validate(1, "not important", "not important")
	authenticated, err := mockValidator.Validate(2, "not important", "not important")

	assert.Errorf(t, err, "provided clientID does not mach active clientID")
	assert.False(t, authenticated)
}

func TestValidateReturnsTrueWhenSessionExistsAndSignatureIsValidAndClientIDMatches(t *testing.T) {
	mockValidator.Validate(1, "not important", "not important")
	authenticated, err := mockValidator.Validate(1, "not important", "not important")

	assert.NoError(t, err)
	assert.True(t, authenticated)
}

func TestCleanupReturnsNoErrorIfSessionIsCleared(t *testing.T) {
	mockValidator.Validate(1, "not important", "not important")
	err := mockValidator.Cleanup("not important")
	_, found, _ := fakeManager.FindSession(1, "not important")
	assert.False(t, found)
	assert.NoError(t, err)
}

func TestCleanupReturnsErrorIfSessionNotExists(t *testing.T) {
	mockManager := &MockSessionManager{}
	mockExtractor := &MockIdentityExtractor{
		identity.FromAddress("deadbeef"),
		nil,
	}
	fakeManager := NewClientMap(mockManager)
	mockValidator := NewValidator(fakeManager, mockExtractor)

	err := mockValidator.Cleanup("nonexistent_session")

	assert.Errorf(t, err, "no underlying session exists: nonexistent_session")
}
