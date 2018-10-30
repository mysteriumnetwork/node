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
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// swagger:model SessionsDTO
type sessionsDTO struct {
	Sessions []sessionDTO `json:"sessions"`
}

// swagger:model SessionDTO
type sessionDTO struct {
	// example: 4cfb0324-daf6-4ad8-448b-e61fe0a1f918
	SessionID string `json:"sessionId"`

	// example: 0x0000000000000000000000000000000000000001
	ProviderID string `json:"providerId"`

	// example: openpvn
	ServiceType string `json:"serviceType"`

	// example: NL
	ProviderCountry string `json:"providerCountry"`

	// example: 2018-10-29 16:22:05
	DateStarted string `json:"dateStarted"`

	// example: 1024
	BytesSent int `json:"bytesSent"`

	// example: 1024
	BytesReceived int `json:"bytesReceived"`

	// duration in seconds
	// example: 120
	Duration int `json:"duration"`
}

type sessionsEndpoint struct {
	sessionStorage connection.SessionStorage
}

// NewSessionsEndpoint creates and returns sessions endpoint
func NewSessionsEndpoint(sessionStorage connection.SessionStorage) *sessionsEndpoint {
	return &sessionsEndpoint{
		sessionStorage: sessionStorage,
	}
}

// swagger:operation GET /sessions Session listSessions
// ---
// summary: Returns sessions history
// description: Returns list of sessions history
// responses:
//   200:
//     description: List of sessions
//     schema:
//       "$ref": "#/definitions/SessionsDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (endpoint *sessionsEndpoint) List(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	sessions, err := endpoint.sessionStorage.GetAll()
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}
	sessionsSerializable := sessionsDTO{Sessions: mapSessions(sessions, sessionToDto)}
	utils.WriteAsJSON(sessionsSerializable, resp)
}

// AddRoutesForSession attaches sessions endpoints to router
func AddRoutesForSession(router *httprouter.Router, sessionsRepository connection.SessionStorage) {
	sessionsEndpoint := NewSessionsEndpoint(sessionsRepository)
	router.GET("/sessions", sessionsEndpoint.List)
}

func sessionToDto(se connection.Session) sessionDTO {
	return sessionDTO{
		SessionID:       string(se.SessionID),
		ProviderID:      string(se.ProviderID.Address),
		ServiceType:     se.ServiceType,
		ProviderCountry: se.ProviderCountry,
		DateStarted:     se.TimeStarted.Format("2018-10-29 16:22:05"),
		BytesSent:       se.DataStats.BytesSent,
		BytesReceived:   se.DataStats.BytesReceived,
		Duration:        se.Duration,
	}
}

func mapSessions(sessions []connection.Session, f func(connection.Session) sessionDTO) []sessionDTO {
	dtoArray := make([]sessionDTO, len(sessions))
	for i, se := range sessions {
		dtoArray[i] = f(se)
	}
	return dtoArray
}
