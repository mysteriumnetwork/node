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

func mockStorage(sessions ...session.Session) *session.StorageMemory {
	storage := session.NewStorageMemory()
	for _, sessionInstance := range sessions {
		storage.Add(sessionInstance)
	}

	return storage
}

func mockValidator(identityToExtract identity.Identity, sessions ...session.Session) *Validator {
	mockExtractor := &MockIdentityExtractor{
		identityToExtract,
		nil,
	}
	return NewValidator(mockStorage(sessions...), mockExtractor)
}

// MockIdentityExtractor mocked identity extractor
type MockIdentityExtractor struct {
	OnExtractReturnIdentity identity.Identity
	OnExtractReturnError    error
}

// Extract returns mocked identity
func (extractor *MockIdentityExtractor) Extract(message []byte, signature identity.Signature) (identity.Identity, error) {
	return extractor.OnExtractReturnIdentity, extractor.OnExtractReturnError
}
