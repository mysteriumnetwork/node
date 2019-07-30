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

	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// swagger:model LocationDTO
type locationResponse struct {
	// IP address
	// example: 1.2.3.4
	IP string `json:"ip"`
	// Autonomous system number
	// example: 62179
	ASN int `json:"asn"`
	// Internet Service Provider name
	// example: Telia Lietuva, AB
	ISP string `json:"isp"`

	// Continent
	// example: EU
	Continent string `json:"continent"`
	// Node Country
	// example: LT
	Country string `json:"country"`
	// Node City
	// example: Vilnius
	City string `json:"city"`

	// User type (data_center, residential, etc.)
	// example: residential
	UserType string `json:"userType"`
	// User type (DEPRECIATED)
	// example: residential
	NodeType string `json:"node_type"`
}

// LocationEndpoint struct represents /location resource and it's subresources
type LocationEndpoint struct {
	locationResolver location.Resolver
}

// NewLocationEndpoint creates and returns location endpoint
func NewLocationEndpoint(locationResolver location.Resolver) *LocationEndpoint {
	return &LocationEndpoint{
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
	currentLocation, err := le.locationResolver.DetectLocation()
	if err != nil {
		utils.SendError(writer, err, http.StatusServiceUnavailable)
		return
	}

	response := locationResponse{
		IP:        currentLocation.IP,
		ASN:       currentLocation.ASN,
		ISP:       currentLocation.ISP,
		Continent: currentLocation.Continent,
		Country:   currentLocation.Country,
		City:      currentLocation.City,
		UserType:  currentLocation.NodeType,
		NodeType:  currentLocation.NodeType,
	}
	utils.WriteAsJSON(response, writer)
}

// AddRoutesForLocation adds location routes to given router
func AddRoutesForLocation(router *httprouter.Router, locationResolver location.Resolver) {
	locationEndpoint := NewLocationEndpoint(locationResolver)
	router.GET("/location", locationEndpoint.GetLocation)
}
