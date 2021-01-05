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

package service

import (
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session"
)

func createAuthHandler(identityToExtract identity.Identity) *authHandler {
	mockExtractor := &mockIdentityExtractor{
		identityToExtract,
		nil,
	}
	mockSessions := &mockSessions{}
	return newAuthHandler(NewClientMap(mockSessions), mockExtractor)
}

func createAuthHandlerWithSession(identityToExtract identity.Identity, sessionInstance *service.Session) *authHandler {
	mockExtractor := &mockIdentityExtractor{
		identityToExtract,
		nil,
	}
	mockSessions := &mockSessions{
		sessionInstance,
		true,
	}
	return newAuthHandler(NewClientMap(mockSessions), mockExtractor)
}

// mockIdentityExtractor mocked identity extractor
type mockIdentityExtractor struct {
	OnExtractReturnIdentity identity.Identity
	OnExtractReturnError    error
}

// Extract returns mocked identity
func (extractor *mockIdentityExtractor) Extract(_ []byte, _ identity.Signature) (identity.Identity, error) {
	return extractor.OnExtractReturnIdentity, extractor.OnExtractReturnError
}

type mockSessions struct {
	OnFindReturnSession *service.Session
	OnFindReturnSuccess bool
}

func (sessions *mockSessions) Add(_ *service.Session) {
}

func (sessions *mockSessions) Find(_ session.ID) (*service.Session, bool) {
	return sessions.OnFindReturnSession, sessions.OnFindReturnSuccess
}

func (sessions *mockSessions) Remove(session.ID) {
	sessions.OnFindReturnSession = nil
	sessions.OnFindReturnSuccess = false
}
