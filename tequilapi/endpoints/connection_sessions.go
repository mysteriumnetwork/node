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
//       "$ref": "#/definitions/ListSessionsResponse"
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

	sessionsDTO := contract.NewSessionListResponse(sessions)
	utils.WriteAsJSON(sessionsDTO, resp)
}

// AddRoutesForConnectionSessions attaches connection sessions endpoints to router
func AddRoutesForConnectionSessions(router *httprouter.Router, sessionStorage connectionSessionStorage) {
	sessionsEndpoint := NewConnectionSessionsEndpoint(sessionStorage)
	router.GET("/connection-sessions", sessionsEndpoint.List)
}
