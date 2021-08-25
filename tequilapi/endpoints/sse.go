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

	"github.com/gin-gonic/gin"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/eventbus"
)

// SSEHandler represents the sse handler
type SSEHandler interface {
	Sub(resp http.ResponseWriter, req *http.Request, params httprouter.Params)
}

// AddRoutesForSSE adds route for sse
func AddRoutesForSSE(e *gin.Engine, stateProvider stateProvider, bus eventbus.EventBus) error {
	sseHandler := NewSSEHandler(stateProvider)
	if err := sseHandler.Subscribe(bus); err != nil {
		return err
	}
	e.GET("/events/state", sseHandler.Sub)
	return nil
}
