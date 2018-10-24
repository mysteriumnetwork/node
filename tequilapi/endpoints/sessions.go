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
	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/storage"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"net/http"
)

type SessionsEndpoint struct {
	storage storage.Storage
}

func NewSessionsEndpoint(storage storage.Storage) *SessionsEndpoint {
	return &SessionsEndpoint{
		storage: storage,
	}
}

func (endpoint *SessionsEndpoint) List(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	sessions := []connection.Session{}
	if err := endpoint.storage.GetAll("all-sessions", &sessions); err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}
	utils.WriteAsJSON(sessions, resp)
}

func AddRoutesForSession(router *httprouter.Router, storage storage.Storage) {
	sessionsEndpoint := NewSessionsEndpoint(storage)
	router.GET("/sessions", sessionsEndpoint.List)
}
