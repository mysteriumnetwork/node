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
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/session"
	"github.com/stretchr/testify/assert"
	"testing"
)

var mockConfig = ""

func TestAuthenticatorReturnsFalseWhenNoSessionFound(t *testing.T) {
	mockManager := &mockSessionManager{}
	mockExtractor := &mockExtractor{}
	validateCredentials := NewSessionValidator(mockManager.FindSession, mockExtractor)

	authenticated, err := validateCredentials("not important", "not important")
	assert.NoError(t, err)
	assert.False(t, authenticated)
}

func TestAuthenticatorReturnsFalseWhenSignatureIsInvalid(t *testing.T) {
	mockManager := mockSessionManager{
		session.Session{
			ID:         session.SessionID("fake-id"),
			Config:     mockConfig,
			ConsumerID: identity.FromAddress("deadbeef"),
		},
		true,
	}
	mockExtractor := &mockExtractor{
		identity.FromAddress("wrongsignature"),
		nil,
	}
	validateCredentials := NewSessionValidator(mockManager.FindSession, mockExtractor)

	authenticated, err := validateCredentials("not important", "not important")
	assert.NoError(t, err)
	assert.False(t, authenticated)
}

func TestAuthenticatorReturnsTrueWhenSessionExistsAndSignatureIsValid(t *testing.T) {
	mockManager := mockSessionManager{
		session.Session{
			ID:         session.SessionID("fake-id"),
			Config:     mockConfig,
			ConsumerID: identity.FromAddress("deadbeef"),
		},
		true,
	}
	mockExtractor := &mockExtractor{
		identity.FromAddress("deadbeef"),
		nil,
	}
	validateCredentials := NewSessionValidator(mockManager.FindSession, mockExtractor)

	authenticated, err := validateCredentials("not important", "not important")
	assert.NoError(t, err)
	assert.True(t, authenticated)

}

type mockSessionManager struct {
	onFindReturnSession session.Session
	onFindReturnSuccess bool
}

func (manager *mockSessionManager) FindSession(sessionId session.SessionID) (session.Session, bool) {
	return manager.onFindReturnSession, manager.onFindReturnSuccess
}

type mockExtractor struct {
	onExtractReturnIdentity identity.Identity
	onExtractReturnError    error
}

func (extractor *mockExtractor) Extract(message []byte, signature identity.Signature) (identity.Identity, error) {
	return extractor.onExtractReturnIdentity, extractor.onExtractReturnError
}
