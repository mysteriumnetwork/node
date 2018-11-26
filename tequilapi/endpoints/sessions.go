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
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// swagger:model SessionsDTO
type SessionsDTO struct {
	Sessions []SessionDTO `json:"sessions"`
}

// swagger:model SessionDTO
type SessionDTO struct {
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
	BytesSent uint64 `json:"bytesSent"`

	// example: 1024
	BytesReceived uint64 `json:"bytesReceived"`

	// duration in seconds
	// example: 120
	Duration uint64 `json:"duration"`

	// example: Completed
	Status string `json:"status"`
}

type sessionsEndpoint struct {
	sessionStorage sessionStorageGet
}

type sessionStorageGet interface {
	GetAll() ([]session.Session, error)
}

// NewSessionsEndpoint creates and returns sessions endpoint
func NewSessionsEndpoint(sessionStorage sessionStorageGet) *sessionsEndpoint {
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
	sessionsSerializable := SessionsDTO{Sessions: mapSessions(sessions, sessionToDto)}
	utils.WriteAsJSON(sessionsSerializable, resp)
}

// AddRoutesForSession attaches sessions endpoints to router
func AddRoutesForSession(router *httprouter.Router, sessionStorage sessionStorageGet) {
	sessionsEndpoint := NewSessionsEndpoint(sessionStorage)
	router.GET("/sessions", sessionsEndpoint.List)
}

func sessionToDto(se session.Session) SessionDTO {
	return SessionDTO{
		SessionID:       string(se.SessionID),
		ProviderID:      string(se.ProviderID.Address),
		ServiceType:     se.ServiceType,
		ProviderCountry: se.ProviderCountry,
		DateStarted:     se.Started.Format("2018-10-29 16:22:05"),
		BytesSent:       se.DataStats.BytesSent,
		BytesReceived:   se.DataStats.BytesReceived,
		Duration:        se.GetDuration(),
		Status:          se.Status.String(),
	}
}

func mapSessions(sessions []session.Session, f func(session.Session) SessionDTO) []SessionDTO {
	dtoArray := make([]SessionDTO, len(sessions))
	for i, se := range sessions {
		dtoArray[i] = f(se)
	}
	return dtoArray
}
