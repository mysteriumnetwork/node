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
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

type healthCheckEndpoint struct {
	startTime       time.Time
	currentTimeFunc func() time.Time
	processNumber   int
}

/*
HealthCheckEndpointFactory creates a structure with single HealthCheck method for healthcheck serving as http,
currentTimeFunc is injected for easier testing
*/
func HealthCheckEndpointFactory(currentTimeFunc func() time.Time, procID func() int) *healthCheckEndpoint {
	startTime := currentTimeFunc()
	return &healthCheckEndpoint{
		startTime,
		currentTimeFunc,
		procID(),
	}
}

// swagger:operation GET /healthcheck Client healthCheck
//
//	---
//	summary: Returns information about client
//	description: Returns health check information about client
//	responses:
//	  200:
//	    description: Health check information
//	    schema:
//	      "$ref": "#/definitions/HealthCheckDTO"
func (hce *healthCheckEndpoint) HealthCheck(c *gin.Context) {
	status := contract.HealthCheckDTO{
		Uptime:  hce.currentTimeFunc().Sub(hce.startTime).String(),
		Process: hce.processNumber,
		Version: metadata.VersionAsString(),
		BuildInfo: contract.BuildInfoDTO{
			Commit:      metadata.BuildCommit,
			Branch:      metadata.BuildBranch,
			BuildNumber: metadata.BuildNumber,
		},
	}
	utils.WriteAsJSON(status, c.Writer)
}
