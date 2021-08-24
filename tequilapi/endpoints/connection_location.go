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

	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/location/locationstate"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

func locationToRes(l locationstate.Location) contract.LocationDTO {
	return contract.LocationDTO{
		IP:        l.IP,
		ASN:       l.ASN,
		ISP:       l.ISP,
		Continent: l.Continent,
		Country:   l.Country,
		City:      l.City,
		IPType:    l.IPType,
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
func (le *ConnectionLocationEndpoint) GetConnectionIP(c *gin.Context) {
	writer := c.Writer
	ipAddress, err := le.ipResolver.GetPublicIP()
	if err != nil {
		utils.SendError(writer, err, http.StatusServiceUnavailable)
		return
	}

	response := contract.IPDTO{
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
func (le *ConnectionLocationEndpoint) GetConnectionLocation(c *gin.Context) {
	writer := c.Writer
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
func (le *ConnectionLocationEndpoint) GetOriginLocation(c *gin.Context) {
	originLocation := le.locationOriginResolver.GetOrigin()

	utils.WriteAsJSON(locationToRes(originLocation), c.Writer)
}

// AddRoutesForConnectionLocation adds connection location routes to given router
func AddRoutesForConnectionLocation(
	ipResolver ip.Resolver,
	locationResolver location.Resolver,
	locationOriginResolver location.OriginResolver,
) func(*gin.Engine) error {

	connectionLocationEndpoint := NewConnectionLocationEndpoint(ipResolver, locationResolver, locationOriginResolver)
	return func(e *gin.Engine) error {
		connGroup := e.Group("/connection")
		{
			connGroup.GET("/ip", connectionLocationEndpoint.GetConnectionIP)
			connGroup.GET("/location", connectionLocationEndpoint.GetConnectionLocation)
		}

		e.GET("/location", connectionLocationEndpoint.GetOriginLocation)
		return nil
	}
}
