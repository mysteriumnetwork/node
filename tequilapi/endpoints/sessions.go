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
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/vcraescu/go-paginator/adapter"
)

type sessionStorage interface {
	List(*session.Filter) ([]session.History, error)
	Stats(*session.Filter) (session.Stats, error)
	StatsByDay(*session.Filter) (map[time.Time]session.Stats, error)
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
// description: Returns list of sessions history filtered by given query
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
	filter := queryToFilter(query.SessionQuery, session.NewFilter())

	pageSize := 50
	if query.PageSize != nil {
		pageSize = *query.PageSize
	}

	page := 1
	if query.Page != nil {
		page = *query.Page
	}

	sessionsAll, err := endpoint.sessionStorage.List(filter)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	var sessions []session.History
	p := utils.NewPaginator(adapter.NewSliceAdapter(sessionsAll), pageSize, page)
	if err := p.Results(&sessions); err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	sessionsDTO := contract.NewSessionListResponse(sessions, p)
	utils.WriteAsJSON(sessionsDTO, resp)
}

// swagger:operation GET /sessions/stats-aggregated Session sessionStatsAggregated
// ---
// summary: Returns sessions stats
// description: Returns aggregated statistics of sessions filtered by given query
// responses:
//   200:
//     description: List of sessions
//     schema:
//       "$ref": "#/definitions/SessionStatsAggregatedResponse"
//   422:
//     description: Parameters validation error
//     schema:
//       "$ref": "#/definitions/ValidationErrorDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (endpoint *sessionsEndpoint) StatsAggregated(resp http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	query, errors := contract.NewSessionQuery(request)
	if errors.HasErrors() {
		utils.SendValidationErrorMessage(resp, errors)
		return
	}
	filter := queryToFilter(query, session.NewFilter())

	stats, err := endpoint.sessionStorage.Stats(filter)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	sessionsDTO := contract.NewSessionStatsAggregatedResponse(stats)
	utils.WriteAsJSON(sessionsDTO, resp)
}

// swagger:operation GET /sessions/stats-daily Session sessionStatsDaily
// ---
// summary: Returns sessions stats
// description: Returns aggregated daily statistics of sessions filtered by given query
// responses:
//   200:
//     description: List of sessions
//     schema:
//       "$ref": "#/definitions/SessionStatsDTO"
//   422:
//     description: Parameters validation error
//     schema:
//       "$ref": "#/definitions/ValidationErrorDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (endpoint *sessionsEndpoint) StatsDaily(resp http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	query, errors := contract.NewSessionQuery(request)
	if errors.HasErrors() {
		utils.SendValidationErrorMessage(resp, errors)
		return
	}
	filter := session.NewFilter().
		SetStartedFrom(time.Now().AddDate(0, 0, -30)).
		SetStartedTo(time.Now())
	filter = queryToFilter(query, filter)

	stats, err := endpoint.sessionStorage.Stats(filter)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	statsDaily, err := endpoint.sessionStorage.StatsByDay(filter)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	sessionsDTO := contract.NewSessionStatsDailyResponse(stats, statsDaily)
	utils.WriteAsJSON(sessionsDTO, resp)
}

// AddRoutesForSessions attaches sessions endpoints to router
func AddRoutesForSessions(router *httprouter.Router, sessionStorage sessionStorage) {
	sessionsEndpoint := NewSessionsEndpoint(sessionStorage)
	router.GET("/sessions", sessionsEndpoint.List)
	router.GET("/sessions/stats-aggregated", sessionsEndpoint.StatsAggregated)
	router.GET("/sessions/stats-daily", sessionsEndpoint.StatsDaily)
}

func queryToFilter(query contract.SessionQuery, filter *session.Filter) *session.Filter {
	if query.DateFrom != nil {
		filter.SetStartedFrom(time.Time(*query.DateFrom).Truncate(24 * time.Hour))
	}
	if query.DateTo != nil {
		filter.SetStartedTo(time.Time(*query.DateTo).Truncate(24 * time.Hour).Add(23 * time.Hour).Add(59 * time.Minute).Add(59 * time.Second))
	}
	if query.Direction != nil {
		filter.SetDirection(*query.Direction)
	}
	if query.ConsumerID != nil {
		filter.SetConsumerID(identity.FromAddress(*query.ConsumerID))
	}
	if query.HermesID != nil {
		filter.SetHermesID(*query.HermesID)
	}
	if query.ProviderID != nil {
		filter.SetProviderID(identity.FromAddress(*query.ProviderID))
	}
	if query.ServiceType != nil {
		filter.SetServiceType(*query.ServiceType)
	}
	if query.Status != nil {
		filter.SetStatus(*query.Status)
	}
	return filter
}
