/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

// swagger:model LocationDTO
type locationResponse struct {
	Original location.Location `json:"original"`
	Current  location.Location `json:"current"`
}

// LocationEndpoint struct represents /location resource and it's subresources
type LocationEndpoint struct {
	manager               connection.Manager
	locationDetector      location.Detector
	originalLocationCache location.Cache
}

// NewLocationEndpoint creates and returns location endpoint
func NewLocationEndpoint(manager connection.Manager, locationDetector location.Detector,
	originalLocationCache location.Cache) *LocationEndpoint {
	return &LocationEndpoint{
		manager:               manager,
		locationDetector:      locationDetector,
		originalLocationCache: originalLocationCache,
	}
}

// GetLocation responds with original and current locations
// swagger:operation GET /location Location getLocation
// ---
// summary: Returns location
// description: Returns original and current locations
// responses:
//   200:
//     description: Original and current locations
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
func (le *LocationEndpoint) GetLocation(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	originalLocation := le.originalLocationCache.Get()

	var currentLocation location.Location
	var err error
	if le.manager.Status().State == connection.Connected {
		currentLocation, err = le.locationDetector.DetectLocation()
		if err != nil {
			utils.SendError(writer, err, http.StatusServiceUnavailable)
			return
		}
	} else {
		currentLocation = originalLocation
	}

	response := locationResponse{
		Original: originalLocation,
		Current:  currentLocation,
	}

	utils.WriteAsJSON(response, writer)
}

// AddRoutesForLocation adds location routes to given router
func AddRoutesForLocation(router *httprouter.Router, manager connection.Manager,
	locationDetector location.Detector, locationCache location.Cache) {

	locationEndpoint := NewLocationEndpoint(manager, locationDetector, locationCache)
	router.GET("/location", locationEndpoint.GetLocation)
}
