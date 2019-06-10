/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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
	"sort"

	"github.com/julienschmidt/httprouter"
	stateEvent "github.com/mysteriumnetwork/node/core/state/event"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// serviceSessionsList defines session list representable as json
// swagger:model ServiceSessionListDTO
type serviceSessionsList struct {
	Sessions []stateEvent.ServiceSession `json:"sessions"`
}

type stateStorage interface {
	GetState() stateEvent.State
}

type serviceSessionsEndpoint struct {
	stateStorage stateStorage
}

// NewServiceSessionsEndpoint creates and returns sessions endpoint
func NewServiceSessionsEndpoint(stateStorage stateStorage) *serviceSessionsEndpoint {
	return &serviceSessionsEndpoint{
		stateStorage: stateStorage,
	}
}

// swagger:operation GET /service/:id/sessions Service serviceSessions
// ---
// summary: Returns current sessions
// description: Returns list of sessions in currently running service
// responses:
//   200:
//     description: List of sessions
//     schema:
//       "$ref": "#/definitions/ServiceSessionListDTO"
func (endpoint *serviceSessionsEndpoint) List(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	sessions := endpoint.stateStorage.GetState().Sessions

	sort.Slice(sessions, func(i, j int) bool { return sessions[i].CreatedAt.Before(sessions[j].CreatedAt) })

	sessionsSerializable := serviceSessionsList{
		Sessions: sessions,
	}
	utils.WriteAsJSON(sessionsSerializable, resp)
}

// AddRoutesForServiceSessions attaches service sessions endpoints to router
func AddRoutesForServiceSessions(router *httprouter.Router, stateStorage stateStorage) {
	sessionsEndpoint := NewServiceSessionsEndpoint(stateStorage)
	router.GET("/service-sessions", sessionsEndpoint.List)
}
