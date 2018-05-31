/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package bytescount

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/server/dto"
	"github.com/mysterium/node/session"
)

// SessionStatsSender sends statistics to server
type SessionStatsSender func(bytesSent, bytesReceived int) error

// NewSessionStatsSender returns new session stats handler, which sends statistics to server
func NewSessionStatsSender(mysteriumClient server.Client, sessionID session.SessionID, providerID identity.Identity, signer identity.Signer, ConsumerCountry string) SessionStatsHandler {
	sessionIDString := string(sessionID)
	return func(sessionStats SessionStats) error {
		return mysteriumClient.SendSessionStats(
			sessionIDString,
			dto.SessionStats{
				BytesSent:       sessionStats.BytesSent,
				BytesReceived:   sessionStats.BytesReceived,
				ProviderID:      providerID.Address,
				ConsumerCountry: ConsumerCountry,
			},
			signer,
		)
	}
}
