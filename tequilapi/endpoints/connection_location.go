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
	"github.com/mysteriumnetwork/node/core/ip"
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
	UserType string `json:"user_type"`
	// User type (DEPRECATED)
	// example: residential
	NodeType string `json:"node_type"`
}

func locationToRes(l location.Location) locationResponse {
	return locationResponse{
		IP:        l.IP,
		ASN:       l.ASN,
		ISP:       l.ISP,
		Continent: l.Continent,
		Country:   l.Country,
		City:      l.City,
		UserType:  l.NodeType,
		NodeType:  l.NodeType,
	}
}

// ConnectionLocationEndpoint struct represents /connection/location resource and it's subresources.
type ConnectionLocationEndpoint struct {
	ipResolver             ip.Resolver
	locationResolver       location.Resolver
	locationOriginResolver location.OriginResolver
}

// NewConnectionLocationEndpoint creates and returns connection location endpoint.
func NewConnectionLocationEndpoint(
	ipResolver ip.Resolver,
	locationResolver location.Resolver,
	locationOriginResolver location.OriginResolver,
) *ConnectionLocationEndpoint {
	return &ConnectionLocationEndpoint{
		ipResolver:             ipResolver,
		locationResolver:       locationResolver,
		locationOriginResolver: locationOriginResolver,
	}
}

// GetConnectionIP responds with current ip, using its ip resolver
// swagger:operation GET /connection/ip Connection getConnectionIP
// ---
// summary: Returns IP address
// description: Returns current public IP address
// responses:
//   200:
//     description: Public IP address
//     schema:
//       "$ref": "#/definitions/IPDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   503:
//     description: Service unavailable
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (le *ConnectionLocationEndpoint) GetConnectionIP(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ipAddress, err := le.ipResolver.GetPublicIP()
	if err != nil {
		utils.SendError(writer, err, http.StatusServiceUnavailable)
		return
	}

	response := ipResponse{
		IP: ipAddress,
	}
	utils.WriteAsJSON(response, writer)
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
	currentLocation, err := le.locationResolver.DetectLocation()
	if err != nil {
		utils.SendError(writer, err, http.StatusServiceUnavailable)
		return
	}

	utils.WriteAsJSON(locationToRes(currentLocation), writer)
}

// GetOriginLocation responds with original locations
// swagger:operation GET /location Location getOriginLocation
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
func (le *ConnectionLocationEndpoint) GetOriginLocation(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	originLocation, err := le.locationOriginResolver.GetOrigin()
	if err != nil {
		utils.SendError(writer, err, http.StatusServiceUnavailable)
		return
	}

	utils.WriteAsJSON(locationToRes(originLocation), writer)
}

// AddRoutesForConnectionLocation adds connection location routes to given router
func AddRoutesForConnectionLocation(
	router *httprouter.Router,
	ipResolver ip.Resolver,
	locationResolver location.Resolver,
	locationOriginResolver location.OriginResolver,
) {

	connectionLocationEndpoint := NewConnectionLocationEndpoint(ipResolver, locationResolver, locationOriginResolver)
	router.GET("/connection/ip", connectionLocationEndpoint.GetConnectionIP)
	router.GET("/connection/location", connectionLocationEndpoint.GetConnectionLocation)
	router.GET("/location", connectionLocationEndpoint.GetOriginLocation)
}
