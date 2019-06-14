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
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session"
)

func mockValidator(identityToExtract identity.Identity) *Validator {
	mockExtractor := &mockIdentityExtractor{
		identityToExtract,
		nil,
	}
	mockSessions := &mockSessions{
		session.Session{},
		false,
	}
	return NewValidator(NewClientMap(mockSessions), mockExtractor)
}

func mockValidatorWithSession(identityToExtract identity.Identity, sessionInstance session.Session) *Validator {
	mockExtractor := &mockIdentityExtractor{
		identityToExtract,
		nil,
	}
	mockSessions := &mockSessions{
		sessionInstance,
		true,
	}
	return NewValidator(NewClientMap(mockSessions), mockExtractor)
}

// mockIdentityExtractor mocked identity extractor
type mockIdentityExtractor struct {
	OnExtractReturnIdentity identity.Identity
	OnExtractReturnError    error
}

// Extract returns mocked identity
func (extractor *mockIdentityExtractor) Extract(message []byte, signature identity.Signature) (identity.Identity, error) {
	return extractor.OnExtractReturnIdentity, extractor.OnExtractReturnError
}

type mockSessions struct {
	OnFindReturnSession session.Session
	OnFindReturnSuccess bool
}

func (sessions *mockSessions) Add(sessionInstance session.Session) {
}

func (sessions *mockSessions) Find(session.ID) (session.Session, bool) {
	return sessions.OnFindReturnSession, sessions.OnFindReturnSuccess
}

func (sessions *mockSessions) Remove(session.ID) {
	sessions.OnFindReturnSession = session.Session{}
	sessions.OnFindReturnSuccess = false
}
