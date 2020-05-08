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
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/server"
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
	ah := new(authHandler)
	ah.Middleware = credentials.NewMiddleware(ah.validate)
	ah.Middleware.ClientsSubscribe(ah.handleClientEvent)
	ah.clientMap = clientMap
	ah.identityExtractor = extractor
	return ah
}

func (ah *authHandler) handleClientEvent(event server.ClientEvent) {
	switch event.EventType {
	case server.Connect:
		ah.clientMap.Add(event.ClientID, session.ID(event.Env["username"]))
	case server.Disconnect:
		ah.clientMap.Remove(event.ClientID)
	}
}

// handleAuthorisation provides glue code for openvpn management interface to validate incoming client login request,
// it expects session id as username, and session signature signed by client as password
func (ah *authHandler) validate(_ int, username, password string) (bool, error) {
	sessionID := session.ID(username)
	currentSession, currentSessionFound := ah.clientMap.GetSession(sessionID)
	if !currentSessionFound {
		log.Warn().Msgf("Possible break-in attempt. No established session exists: %s", sessionID)
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

	return true, nil
}
