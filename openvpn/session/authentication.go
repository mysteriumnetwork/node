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
	client_auth "github.com/mysterium/node/openvpn/middlewares/client/auth"
	server_auth "github.com/mysterium/node/openvpn/middlewares/server/auth"
	"github.com/mysterium/node/session"
)

const sessionSignaturePrefix = "MystVpnSessionId:"

// SignatureCredentialsProvider returns session id as username and id signed with given signer as password
func SignatureCredentialsProvider(id session.SessionID, signer identity.Signer) client_auth.CredentialsProvider {
	return func() (string, string, error) {
		signature, err := signer.Sign([]byte(sessionSignaturePrefix + id))
		return string(id), signature.Base64(), err
	}
}

type sessionFinder func(session session.SessionID) (session.Session, bool)

// NewSessionValidator provides glue code for openvpn management interface to validate incoming client login request,
// it expects session id as username, and session signature signed by client as password
func NewSessionValidator(findSession sessionFinder, extractor identity.Extractor) server_auth.CredentialsChecker {
	return func(sessionString, signatureString string) (bool, error) {
		sessionId := session.SessionID(sessionString)
		currentSession, found := findSession(sessionId)
		if !found {
			return false, nil
		}

		signature := identity.SignatureBase64(signatureString)
		extractedIdentity, err := extractor.Extract([]byte(sessionSignaturePrefix+sessionString), signature)
		if err != nil {
			return false, err
		}
		return currentSession.ConsumerID == extractedIdentity, nil
	}
}
