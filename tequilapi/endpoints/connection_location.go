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
	manager               connection.Manager
	locationDetector      location.Detector
	originalLocationCache location.Cache
}

// NewConnectionLocationEndpoint creates and returns connection location endpoint.
func NewConnectionLocationEndpoint(manager connection.Manager, locationDetector location.Detector,
	originalLocationCache location.Cache) *LocationEndpoint {
	return &LocationEndpoint{
		manager:               manager,
		locationDetector:      locationDetector,
		originalLocationCache: originalLocationCache,
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
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   503:
//     description: Service unavailable
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (le *ConnectionLocationEndpoint) GetConnectionLocation(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	connectionLocation := locationResponse{
		IP:  "1.2.3.4",
		ASN: "62179",
		ISP: "Telia Lietuva, AB",

		Continent: "EU",
		Country:   "LT",
		City:      "Vilnius",

		NodeType: "residential",
	}

	utils.WriteAsJSON(connectionLocation, writer)
}

// AddRoutesForConnectionLocation adds connection location routes to given router
func AddRoutesForConnectionLocation(router *httprouter.Router, manager connection.Manager,
	locationDetector location.Detector, locationCache location.Cache) {

	connectionLocationEndpoint := NewConnectionLocationEndpoint(manager, locationDetector, locationCache)
	router.GET("/connection/location", connectionLocationEndpoint.GetLocation)
}
