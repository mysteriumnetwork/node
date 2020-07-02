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
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

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
//       "$ref": "#/definitions/ListConnectionSessionsResponse"
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

	sessionsSerializable := contract.ListConnectionSessionsResponse{Sessions: mapConnectionSessions(sessions, connectionSessionToDto)}
	utils.WriteAsJSON(sessionsSerializable, resp)
}

// AddRoutesForConnectionSessions attaches connection sessions endpoints to router
func AddRoutesForConnectionSessions(router *httprouter.Router, sessionStorage connectionSessionStorage) {
	sessionsEndpoint := NewConnectionSessionsEndpoint(sessionStorage)
	router.GET("/connection-sessions", sessionsEndpoint.List)
}

func connectionSessionToDto(se session.History) contract.ConnectionSessionDTO {
	return contract.ConnectionSessionDTO{
		SessionID:       string(se.SessionID),
		ConsumerID:      se.ConsumerID.Address,
		AccountantID:    se.AccountantID,
		ProviderID:      se.ProviderID.Address,
		ServiceType:     se.ServiceType,
		ProviderCountry: se.ProviderCountry,
		DateStarted:     se.Started.Format(time.RFC3339),
		BytesSent:       se.DataSent,
		BytesReceived:   se.DataReceived,
		Duration:        uint64(se.GetDuration().Seconds()),
		TokensSpent:     se.Tokens,
		Status:          se.Status,
	}
}

func mapConnectionSessions(sessions []session.History, f func(session.History) contract.ConnectionSessionDTO) []contract.ConnectionSessionDTO {
	dtoArray := make([]contract.ConnectionSessionDTO, len(sessions))
	for i, se := range sessions {
		dtoArray[i] = f(se)
	}
	return dtoArray
}
