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
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/mysteriumnetwork/go-rest/apierror"
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
		Region:    l.Region,
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
//
//	---
//	summary: Returns IP address
//	description: Returns current public IP address
//	responses:
//	  200:
//	    description: Public IP address
//	    schema:
//	      "$ref": "#/definitions/IPDTO"
//	  503:
//	    description: Service unavailable
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (le *ConnectionLocationEndpoint) GetConnectionIP(c *gin.Context) {
	ipAddress, err := le.ipResolver.GetPublicIP()
	if err != nil {
		c.Error(apierror.ServiceUnavailable())
		return
	}

	response := contract.IPDTO{
		IP: ipAddress,
	}
	utils.WriteAsJSON(response, c.Writer)
}

// GetProxyIP responds with proxy ip, using its ip resolver
// swagger:operation GET /connection/proxy/ip Connection getProxyIP
//
//	---
//	summary: Returns IP address
//	description: Returns proxy public IP address
//	responses:
//	  200:
//	    description: Public IP address
//	    schema:
//	      "$ref": "#/definitions/IPDTO"
//	  503:
//	    description: Service unavailable
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (le *ConnectionLocationEndpoint) GetProxyIP(c *gin.Context) {
	n, _ := strconv.Atoi(c.Query("port"))
	ipAddress, err := le.ipResolver.GetProxyIP(n)
	if err != nil {
		c.Error(apierror.ServiceUnavailable())
		return
	}

	response := contract.IPDTO{
		IP: ipAddress,
	}
	utils.WriteAsJSON(response, c.Writer)
}

// GetConnectionLocation responds with current connection location
// swagger:operation GET /connection/location Connection getConnectionLocation
//
//	---
//	summary: Returns connection location
//	description: Returns connection locations
//	responses:
//	  200:
//	    description: Connection locations
//	    schema:
//	      "$ref": "#/definitions/LocationDTO"
//	  503:
//	    description: Service unavailable
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (le *ConnectionLocationEndpoint) GetConnectionLocation(c *gin.Context) {
	currentLocation, err := le.locationResolver.DetectLocation()
	if err != nil {
		c.Error(apierror.ServiceUnavailable())
		return
	}
	utils.WriteAsJSON(locationToRes(currentLocation), c.Writer)
}

// GetProxyLocation responds with proxy connection location
// swagger:operation GET /connection/proxy/location Connection getProxyLocation
//
//	---
//	summary: Returns proxy connection location
//	description: Returns proxy connection locations
//	responses:
//	  200:
//	    description: Proxy connection locations
//	    schema:
//	      "$ref": "#/definitions/LocationDTO"
//	  503:
//	    description: Service unavailable
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (le *ConnectionLocationEndpoint) GetProxyLocation(c *gin.Context) {
	p, _ := strconv.Atoi(c.Query("port"))
	currentLocation, err := le.locationResolver.DetectProxyLocation(p)
	if err != nil {
		c.Error(apierror.ServiceUnavailable())
		return
	}
	utils.WriteAsJSON(locationToRes(currentLocation), c.Writer)
}

// GetOriginLocation responds with original locations
// swagger:operation GET /location Location getOriginLocation
//
//	---
//	summary: Returns original location
//	description: Returns original locations
//	responses:
//	  200:
//	    description: Original locations
//	    schema:
//	      "$ref": "#/definitions/LocationDTO"
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
			connGroup.GET("/proxy/ip", connectionLocationEndpoint.GetProxyIP)
			connGroup.GET("/proxy/location", connectionLocationEndpoint.GetProxyLocation)
			connGroup.GET("/location", connectionLocationEndpoint.GetConnectionLocation)
		}

		e.GET("/location", connectionLocationEndpoint.GetOriginLocation)
		return nil
	}
}
