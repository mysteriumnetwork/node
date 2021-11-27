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

	"github.com/gin-gonic/gin"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/strfmt/conv"
	"github.com/mysteriumnetwork/node/consumer/session"
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
func (endpoint *sessionsEndpoint) List(c *gin.Context) {
	resp := c.Writer
	request := c.Request

	query := contract.NewSessionListQuery()
	if errors := query.Bind(request); errors.HasErrors() {
		utils.SendValidationErrorMessage(resp, errors)
		return
	}

	sessionsAll, err := endpoint.sessionStorage.List(query.ToFilter())
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	var sessions []session.History
	p := utils.NewPaginator(adapter.NewSliceAdapter(sessionsAll), query.PageSize, query.Page)
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
func (endpoint *sessionsEndpoint) StatsAggregated(c *gin.Context) {
	resp := c.Writer
	request := c.Request

	query := contract.NewSessionQuery()
	if errors := query.Bind(request); errors.HasErrors() {
		utils.SendValidationErrorMessage(resp, errors)
		return
	}

	stats, err := endpoint.sessionStorage.Stats(query.ToFilter())
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
// description: Returns aggregated daily statistics of sessions filtered by given query (date_from=<now -30d> and date_to=<now> by default)
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
func (endpoint *sessionsEndpoint) StatsDaily(c *gin.Context) {
	resp := c.Writer
	request := c.Request

	query := contract.SessionQuery{
		DateFrom: conv.Date(strfmt.Date(time.Now().UTC().AddDate(0, 0, -30))),
		DateTo:   conv.Date(strfmt.Date(time.Now().UTC())),
	}
	if errors := query.Bind(request); errors.HasErrors() {
		utils.SendValidationErrorMessage(resp, errors)
		return
	}

	filter := query.ToFilter()
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
func AddRoutesForSessions(sessionStorage sessionStorage) func(*gin.Engine) error {
	sessionsEndpoint := NewSessionsEndpoint(sessionStorage)
	return func(e *gin.Engine) error {
		g := e.Group("/sessions")
		{
			g.GET("", sessionsEndpoint.List)
			g.GET("/stats-aggregated", sessionsEndpoint.StatsAggregated)
			g.GET("/stats-daily", sessionsEndpoint.StatsDaily)
		}
		return nil
	}
}
