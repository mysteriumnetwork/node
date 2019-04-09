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
	// IP address
	// example: 1.2.3.4
	IP string
	// Autonomous system number
	// example: 62179
	ASN string
	// Internet Service Provider name
	// example: Telia Lietuva, AB
	ISP string

	// Continent
	// example: EU
	Continent string
	// Node Country
	// example: LT
	Country string
	// Node City
	// example: Vilnius
	City string

	// Node type
	// example: residential
	NodeType string
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

// GetLocation responds with original locations
// swagger:operation GET /location Location getLocation
// ---
// summary: Returns original location
// description: Returns original locations
// responses:
//   200:
//     description: Original locations
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
	currentLocation := locationResponse{
		IP:  "1.2.3.4",
		ASN: "62179",
		ISP: "Telia Lietuva, AB",

		Continent: "EU",
		Country:   "LT",
		City:      "Vilnius",

		NodeType: "residential",
	}

	utils.WriteAsJSON(currentLocation, writer)
}

// GetLocationByIP responds with requested locations
// swagger:operation GET /location/:ip Location getLocationByIP
// ---
// summary: Returns requested location
// description: Returns requested locations
// responses:
//   200:
//     description: Requested locations
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
func (le *LocationEndpoint) GetLocationByIP(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	currentLocation := locationResponse{
		IP:  params.ByName("ip"),
		ASN: "62179",
		ISP: "Telia Lietuva, AB",

		Continent: "EU",
		Country:   "LT",
		City:      "Vilnius",

		NodeType: "residential",
	}

	utils.WriteAsJSON(currentLocation, writer)
}

// AddRoutesForLocation adds location routes to given router
func AddRoutesForLocation(router *httprouter.Router, manager connection.Manager,
	locationDetector location.Detector, locationCache location.Cache) {

	locationEndpoint := NewLocationEndpoint(manager, locationDetector, locationCache)
	router.GET("/location", locationEndpoint.GetLocation)
	router.GET("/location/:ip", locationEndpoint.GetLocationByIP)
}
