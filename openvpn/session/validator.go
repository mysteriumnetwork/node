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
	"errors"

	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/session"
)

// SignaturePrefix is used to prefix with each session string before calculating signature or extracting identity
const SignaturePrefix = "MystVpnSessionId:"

// Validator structure that keeps attributes needed Validator operations
type Validator struct {
	sessionManager    *manager
	identityExtractor identity.Extractor
}

// NewValidator return Validator instance
func NewValidator(m *manager, extractor identity.Extractor) *Validator {
	return &Validator{
		sessionManager:    m,
		identityExtractor: extractor,
	}
}

// Validate provides glue code for openvpn management interface to validate incoming client login request,
// it expects session id as username, and session signature signed by client as password
func (v *Validator) Validate(clientID int, sessionString, signatureString string) (bool, error) {
	sessionID := session.SessionID(sessionString)
	currentSession, found, err := v.sessionManager.FindSession(clientID, sessionID)

	if err != nil {
		return false, err
	}

	if !found {
		v.sessionManager.UpdateSession(clientID, sessionID)
	}

	signature := identity.SignatureBase64(signatureString)
	extractedIdentity, err := v.identityExtractor.Extract([]byte(SignaturePrefix+sessionString), signature)
	if err != nil {
		return false, err
	}
	return currentSession.ConsumerID == extractedIdentity, nil
}

// Cleanup removes session from underlying session managers
func (v *Validator) Cleanup(sessionString string) error {
	sessionID := session.SessionID(sessionString)
	_, found := v.sessionManager.sessionManager.FindSession(sessionID)

	if !found {
		return errors.New("no underlying session exists: " + sessionString)
	}

	v.sessionManager.RemoveSession(sessionID)
	return nil
}
