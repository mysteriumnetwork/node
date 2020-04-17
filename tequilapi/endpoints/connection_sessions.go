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

package endpoints

import (
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/consumer/session"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// connectionSessionsList defines session list representable as json
// swagger:model ConnectionSessionListDTO
type connectionSessionsList struct {
	Sessions []connectionSession `json:"sessions"`
}

// connectionSession represents the session object
// swagger:model ConnectionSessionDTO
type connectionSession struct {
	// example: 4cfb0324-daf6-4ad8-448b-e61fe0a1f918
	SessionID string `json:"session_id"`

	// example: 0x0000000000000000000000000000000000000001
	ConsumerID string `json:"consumer_id"`

	// example: 0x0000000000000000000000000000000000000001
	AccountantID string `json:"accountant_id"`

	// example: 0x0000000000000000000000000000000000000001
	ProviderID string `json:"provider_id"`

	// example: openvpn
	ServiceType string `json:"service_type"`

	// example: NL
	ProviderCountry string `json:"provider_country"`

	// example: 2018-10-29 16:22:05
	DateStarted string `json:"date_started"`

	// example: 1024
	BytesSent uint64 `json:"bytes_sent"`

	// example: 1024
	BytesReceived uint64 `json:"bytes_received"`

	// duration in seconds
	// example: 120
	Duration uint64 `json:"duration"`

	// example: 500000
	TokensSpent uint64 `json:"tokens_spent"`

	// example: Completed
	Status string `json:"status"`
}

type connectionSessionStorage interface {
	GetAll() ([]session.History, error)
}

type connectionSessionsEndpoint struct {
	sessionStorage connectionSessionStorage
}

// NewConnectionSessionsEndpoint creates and returns sessions endpoint
func NewConnectionSessionsEndpoint(sessionStorage connectionSessionStorage) *connectionSessionsEndpoint {
	return &connectionSessionsEndpoint{
		sessionStorage: sessionStorage,
	}
}

// swagger:operation GET /connection-sessions Connection connectionSessions
// ---
// summary: Returns sessions history
// description: Returns list of sessions history
// responses:
//   200:
//     description: List of sessions
//     schema:
//       "$ref": "#/definitions/ConnectionSessionListDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (endpoint *connectionSessionsEndpoint) List(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	sessions, err := endpoint.sessionStorage.GetAll()
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	sessionsSerializable := connectionSessionsList{Sessions: mapConnectionSessions(sessions, connectionSessionToDto)}
	utils.WriteAsJSON(sessionsSerializable, resp)
}

// AddRoutesForConnectionSessions attaches connection sessions endpoints to router
func AddRoutesForConnectionSessions(router *httprouter.Router, sessionStorage connectionSessionStorage) {
	sessionsEndpoint := NewConnectionSessionsEndpoint(sessionStorage)
	router.GET("/connection-sessions", sessionsEndpoint.List)
}

func connectionSessionToDto(se session.History) connectionSession {
	return connectionSession{
		SessionID:       string(se.SessionID),
		ConsumerID:      se.ConsumerID.Address,
		AccountantID:    se.AccountantID,
		ProviderID:      se.ProviderID.Address,
		ServiceType:     se.ServiceType,
		ProviderCountry: se.ProviderCountry,
		DateStarted:     se.Started.Format(time.RFC3339),
		BytesSent:       se.DataStats.BytesSent,
		BytesReceived:   se.DataStats.BytesReceived,
		Duration:        uint64(se.GetDuration().Seconds()),
		TokensSpent:     se.Invoice.AgreementTotal,
		Status:          se.Status,
	}
}

func mapConnectionSessions(sessions []session.History, f func(session.History) connectionSession) []connectionSession {
	dtoArray := make([]connectionSession, len(sessions))
	for i, se := range sessions {
		dtoArray[i] = f(se)
	}
	return dtoArray
}
