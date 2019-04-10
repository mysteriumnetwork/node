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
	"net"
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
	ASN int
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
	UserType string
}

// LocationEndpoint struct represents /location resource and it's subresources
type LocationEndpoint struct {
	manager          connection.Manager
	locationResolver location.Resolver
}

// NewLocationEndpoint creates and returns location endpoint
func NewLocationEndpoint(manager connection.Manager, locationResolver location.Resolver) *LocationEndpoint {
	return &LocationEndpoint{
		manager:          manager,
		locationResolver: locationResolver,
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
//   503:
//     description: Service unavailable
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (le *LocationEndpoint) GetLocation(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	currentLocation, err := le.locationResolver.DetectLocation(nil)
	if err != nil {
		utils.SendError(writer, err, http.StatusServiceUnavailable)
		return
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
//   503:
//     description: Service unavailable
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (le *LocationEndpoint) GetLocationByIP(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ip := params.ByName("ip")
	location, err := le.locationResolver.DetectLocation(net.ParseIP(ip))
	if err != nil {
		utils.SendError(writer, err, http.StatusServiceUnavailable)
		return
	}

	utils.WriteAsJSON(location, writer)
}

// AddRoutesForLocation adds location routes to given router
func AddRoutesForLocation(router *httprouter.Router, manager connection.Manager, locationResolver location.Resolver) {
	locationEndpoint := NewLocationEndpoint(manager, locationResolver)
	router.GET("/location", locationEndpoint.GetLocation)
	router.GET("/location/:ip", locationEndpoint.GetLocationByIP)
}
