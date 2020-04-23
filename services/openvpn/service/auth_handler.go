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
	"github.com/mysteriumnetwork/go-openvpn/openvpn/management"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/server/auth"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/session"
)

// authHandler authorises incoming Openvpn clients
type authHandler struct {
	management.Middleware

	clientMap         *clientMap
	identityExtractor identity.Extractor
}

// newAuthHandler return authHandler instance
func newAuthHandler(clientMap *clientMap, extractor identity.Extractor) *authHandler {
	ah := &authHandler{
		clientMap:         clientMap,
		identityExtractor: extractor,
	}
	ah.Middleware = auth.NewMiddleware(ah.validate)
	return ah
}

// handleAuthorisation provides glue code for openvpn management interface to validate incoming client login request,
// it expects session id as username, and session signature signed by client as password
func (ah *authHandler) validate(clientID int, username, password string) (bool, error) {
	sessionID := session.ID(username)
	currentSession, found, err := ah.clientMap.FindClientSession(clientID, sessionID)
	if err != nil {
		return false, err
	}

	if !found {
		ah.clientMap.UpdateClientSession(clientID, sessionID)
	}

	signature := identity.SignatureBase64(password)
	extractedIdentity, err := ah.identityExtractor.Extract([]byte(openvpn.AuthSignaturePrefix+username), signature)
	if err != nil {
		return false, err
	}
	return currentSession.ConsumerID == extractedIdentity, nil
}
