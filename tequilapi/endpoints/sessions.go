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
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/consumer/session"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/vcraescu/go-paginator"
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
// parameters:
//   - in: query
//     name: create_from
//     description: Created date to filter the sessions from this date. Formatted in RFC3339 e.g. 2020-07-01T00:00:00Z.
//     type: string
//   - in: query
//     name: create_to
//     description: Created date to filter the sessions until this date. Formatted in RFC3339 e.g. 2020-07-01T00:00:00Z.
//     type: string
//   - in: query
//     name: direction
//     description: Direction to filter the sessions by. Possible values are "Provider", "Consumed".
//     type: string
//   - in: query
//     name: service_type
//     description: Service type to filter the sessions by.
//     type: string
//   - in: query
//     name: status
//     description: Status to filter the sessions by. Possible values are "New", "Completed".
//     type: string
//   - in: query
//     name: page
//     description: Page to filter the sessions by.
//     type: string
// responses:
//   200:
//     description: List of sessions
//     schema:
//       "$ref": "#/definitions/ListSessionsResponse"
//   400:
//     description: Bad request
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (endpoint *sessionsEndpoint) List(resp http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	query := session.NewQuery()

	from := time.Now().AddDate(0, 0, -30)
	if fromStr := request.URL.Query().Get("create_from"); fromStr != "" {
		var err error
		if from, err = time.Parse(time.RFC3339, fromStr); err != nil {
			utils.SendError(resp, err, http.StatusBadRequest)
			return
		}
	}
	query.FilterFrom(from)

	to := time.Now()
	if toStr := request.URL.Query().Get("created_to"); toStr != "" {
		var err error
		if to, err = time.Parse(time.RFC3339, toStr); err != nil {
			utils.SendError(resp, err, http.StatusBadRequest)
			return
		}
	}
	query.FilterTo(to)

	if direction := request.URL.Query().Get("direction"); direction != "" {
		query.FilterDirection(direction)
	}
	if serviceType := request.URL.Query().Get("service_type"); serviceType != "" {
		query.FilterServiceType(serviceType)
	}
	if status := request.URL.Query().Get("status"); status != "" {
		query.FilterStatus(status)
	}

	page := 1
	if pageStr := request.URL.Query().Get("page"); pageStr != "" {
		var err error
		if page, err = strconv.Atoi(pageStr); err != nil {
			utils.SendError(resp, err, http.StatusBadRequest)
			return
		}
	}

	pageSize := 50
	if pageSizeStr := request.URL.Query().Get("page_size"); pageSizeStr != "" {
		var err error
		if pageSize, err = strconv.Atoi(pageSizeStr); err != nil {
			utils.SendError(resp, err, http.StatusBadRequest)
			return
		}
	}

	if err := endpoint.sessionStorage.Query(query.FetchSessions().FetchStats().FetchStatsByDay()); err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	var sessions []session.History
	p := paginator.New(adapter.NewSliceAdapter(query.Sessions), pageSize)
	p.SetPage(page)
	if err := p.Results(&sessions); err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	sessionsDTO := contract.NewSessionListResponse(sessions, &p, query.Stats, query.StatsByDay)
	utils.WriteAsJSON(sessionsDTO, resp)
}

// AddRoutesForSessions attaches sessions endpoints to router
func AddRoutesForSessions(router *httprouter.Router, sessionStorage sessionStorage) {
	sessionsEndpoint := NewSessionsEndpoint(sessionStorage)
	router.GET("/sessions", sessionsEndpoint.List)
}
