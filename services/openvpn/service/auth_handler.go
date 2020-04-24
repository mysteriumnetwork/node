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
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/server/credentials"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/session"
	"github.com/rs/zerolog/log"
)

// authHandler authorizes incoming clients by registering a callback to check auth primitives.
// username - provider's sessionId
// password - consumer's identity signature
type authHandler struct {
	*credentials.Middleware

	clientMap         *clientMap
	identityExtractor identity.Extractor
}

// newAuthHandler return authHandler instance
func newAuthHandler(clientMap *clientMap, extractor identity.Extractor) *authHandler {
	ah := &authHandler{
		clientMap:         clientMap,
		identityExtractor: extractor,
	}
	ah.Middleware = credentials.NewMiddleware(ah.validate)
	return ah
}

// handleAuthorisation provides glue code for openvpn management interface to validate incoming client login request,
// it expects session id as username, and session signature signed by client as password
func (ah *authHandler) validate(clientID int, username, password string) (bool, error) {
	sessionID := session.ID(username)
	currentSession, currestSessionFound := ah.clientMap.GetSession(sessionID)
	if !currestSessionFound {
		log.Warn().Msgf("Possible break-in attempt. No established session exists: %s", sessionID)
		return false, nil
	}

	if currentClientID, exist := ah.clientMap.GetSessionClient(sessionID); exist && clientID != currentClientID {
		log.Warn().Msgf("Possible break-in attempt. Session %s already used by another client %d", sessionID, currentClientID)
		return false, nil
	}

	signature := identity.SignatureBase64(password)
	signerID, err := ah.identityExtractor.Extract([]byte(openvpn.AuthSignaturePrefix+username), signature)
	if err != nil {
		log.Warn().Err(err).Msgf("Possible break-in attempt. Invalid session %s signature", sessionID)
		return false, nil
	}
	if signerID != currentSession.ConsumerID {
		log.Warn().Msgf("Possible break-in attempt. Invalid session %s consumer %s", sessionID, signerID)
		return false, nil
	}

	ah.clientMap.AssignSessionClient(sessionID, clientID)
	return true, nil
}
