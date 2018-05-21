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
)


const mockedVPNConfig = "config_string"

type mockedConfigProvider func() string

func (mcp mockedConfigProvider) ProvideServiceConfig() (session.ServiceConfiguration, error) {
	return mcp(), nil
}

func provideMockedVPNConfig() string {
	return mockedVPNConfig
}

// MockIdentityExtractor mocked identity extractor
type MockIdentityExtractor struct {
	OnExtractReturnIdentity identity.Identity
	OnExtractReturnError    error
}

// MockSessionManager mocked session manager
type MockSessionManager struct {
	OnFindReturnSession session.Session
	OnFindReturnSuccess bool
}

// Create creates mocked session instance
func (manager *MockSessionManager) Create(peerID identity.Identity) (sessionInstance session.Session, err error) {
	return session.Session{}, nil
}

// FindSession returns mocked session
func (manager *MockSessionManager) FindSession(sessionID session.SessionID) (session.Session, bool) {
	return manager.OnFindReturnSession, manager.OnFindReturnSuccess
}

// Extract returns mocked identity
func (extractor *MockIdentityExtractor) Extract(message []byte, signature identity.Signature) (identity.Identity, error) {
	return extractor.OnExtractReturnIdentity, extractor.OnExtractReturnError
}

// RemoveSession stubbed mock method to satisfy interface
func (manager *MockSessionManager) RemoveSession(sessionID session.SessionID) {
	// stub
}
