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
	"testing"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session"
	"github.com/stretchr/testify/assert"
)

const (
	sessionExistingString = "fake-id"
)

var (
	identityExisting = identity.FromAddress("deadbeef")
	sessionExisting  = session.Session{
		ID:         session.ID(sessionExistingString),
		ConsumerID: identityExisting,
	}
)

func TestValidateReturnsFalseWhenNoSessionFound(t *testing.T) {
	validator := mockValidator(identity.Identity{})

	authenticated, err := validator.Validate(1, "not important", "not important")

	assert.Errorf(t, err, "no underlying session exists, possible break-in attempt")
	assert.False(t, authenticated)
}

func TestValidateReturnsFalseWhenSignatureIsInvalid(t *testing.T) {
	validator := mockValidatorWithSession(identity.FromAddress("wrongsignature"), sessionExisting)

	authenticated, err := validator.Validate(1, sessionExistingString, "not important")

	assert.NoError(t, err)
	assert.False(t, authenticated)
}

func TestValidateReturnsTrueWhenSessionExistsAndSignatureIsValid(t *testing.T) {
	validator := mockValidatorWithSession(identityExisting, sessionExisting)

	authenticated, err := validator.Validate(1, sessionExistingString, "not important")

	assert.NoError(t, err)
	assert.True(t, authenticated)
}

func TestValidateReturnsTrueWhenSessionExistsAndSignatureIsValidAndClientIDDiffers(t *testing.T) {
	validator := mockValidatorWithSession(identityExisting, sessionExisting)

	validator.Validate(1, sessionExistingString, "not important")
	authenticated, err := validator.Validate(2, sessionExistingString, "not important")

	assert.NoError(t, err)
	assert.True(t, authenticated)
}

func TestValidateReturnsTrueWhenSessionExistsAndSignatureIsValidAndClientIDMatches(t *testing.T) {
	validator := mockValidatorWithSession(identityExisting, sessionExisting)

	validator.Validate(1, sessionExistingString, "not important")
	authenticated, err := validator.Validate(1, sessionExistingString, "not important")

	assert.NoError(t, err)
	assert.True(t, authenticated)
}

func TestCleanupReturnsNoErrorIfSessionIsCleared(t *testing.T) {
	validator := mockValidatorWithSession(identityExisting, sessionExisting)

	validator.Validate(1, sessionExistingString, "not important")
	err := validator.Cleanup(sessionExistingString)

	_, found, _ := validator.clientMap.FindClientSession(1, "not important")
	assert.False(t, found)
	assert.NoError(t, err)
}

func TestCleanupReturnsErrorIfSessionNotExists(t *testing.T) {
	validator := mockValidator(identityExisting)

	err := validator.Cleanup("nonexistent_session")

	assert.Errorf(t, err, "no underlying session exists: nonexistent_session")
}
