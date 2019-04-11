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

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// ConnectionLocationEndpoint struct represents /connection/location resource and it's subresources.
type ConnectionLocationEndpoint struct {
	manager          connection.Manager
	locationResolver location.Resolver
}

// NewConnectionLocationEndpoint creates and returns connection location endpoint.
func NewConnectionLocationEndpoint(manager connection.Manager, locationResolver location.Resolver) *ConnectionLocationEndpoint {
	return &ConnectionLocationEndpoint{
		manager:          manager,
		locationResolver: locationResolver,
	}
}

// GetConnectionLocation responds with current connection location
// swagger:operation GET /connection/location Connection getConnectionLocation
// ---
// summary: Returns connection location
// description: Returns connection locations
// responses:
//   200:
//     description: Connection locations
//     schema:
//       "$ref": "#/definitions/LocationDTO"
//   503:
//     description: Service unavailable
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (le *ConnectionLocationEndpoint) GetConnectionLocation(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	if le.manager.Status().State != connection.Connected {
		utils.SendErrorMessage(writer, "Connection is not connected", http.StatusServiceUnavailable)
		return
	}

	currentLocation, err := le.locationResolver.DetectLocation(nil)
	if err != nil {
		utils.SendError(writer, err, http.StatusServiceUnavailable)
		return
	}

	utils.WriteAsJSON(currentLocation, writer)
}

// AddRoutesForConnectionLocation adds connection location routes to given router
func AddRoutesForConnectionLocation(router *httprouter.Router, manager connection.Manager,
	locationResolver location.Resolver) {

	connectionLocationEndpoint := NewConnectionLocationEndpoint(manager, locationResolver)
	router.GET("/connection/location", connectionLocationEndpoint.GetConnectionLocation)
}
