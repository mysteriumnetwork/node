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
	"github.com/vcraescu/go-paginator/adapter"
)

type sessionStorage interface {
	Query(*session.Query) error
}

type sessionsEndpoint struct {
	sessionStorage sessionStorage
}

// NewSessionsEndpoint creates and returns sessions endpoint
func NewSessionsEndpoint(sessionStorage sessionStorage) *sessionsEndpoint {
	return &sessionsEndpoint{
		sessionStorage: sessionStorage,
	}
}

// swagger:operation GET /sessions Session sessionList
// ---
// summary: Returns sessions history
// description: Returns list of sessions history
// responses:
//   200:
//     description: List of sessions
//     schema:
//       "$ref": "#/definitions/SessionListResponse"
//   422:
//     description: Parameters validation error
//     schema:
//       "$ref": "#/definitions/ValidationErrorDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (endpoint *sessionsEndpoint) List(resp http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	query, errors := contract.NewSessionListQuery(request)
	if errors.HasErrors() {
		utils.SendValidationErrorMessage(resp, errors)
		return
	}

	filter := session.NewQuery()
	filter.SetStartedFrom(time.Now().AddDate(0, 0, -30))
	if query.DateFrom != nil {
		filter.SetStartedFrom(*query.DateFrom)
	}
	filter.SetStartedTo(time.Now())
	if query.DateTo != nil {
		filter.SetStartedTo(*query.DateTo)
	}
	filter.Direction = query.Direction
	filter.ServiceType = query.ServiceType
	filter.Status = query.Status

	pageSize := 50
	if query.PageSize != nil {
		pageSize = *query.PageSize
	}

	page := 1
	if query.Page != nil {
		page = *query.Page
	}

	if err := endpoint.sessionStorage.Query(filter.FetchSessions().FetchStats().FetchStatsByDay()); err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	var sessions []session.History
	p := utils.NewPaginator(adapter.NewSliceAdapter(filter.Sessions), pageSize, page)
	if err := p.Results(&sessions); err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	sessionsDTO := contract.NewSessionListResponse(sessions, p, filter.Stats, filter.StatsByDay)
	utils.WriteAsJSON(sessionsDTO, resp)
}

// AddRoutesForSessions attaches sessions endpoints to router
func AddRoutesForSessions(router *httprouter.Router, sessionStorage sessionStorage) {
	sessionsEndpoint := NewSessionsEndpoint(sessionStorage)
	router.GET("/sessions", sessionsEndpoint.List)
}
