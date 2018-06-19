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
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/metadata"
	"github.com/mysterium/node/tequilapi/utils"
	"net/http"
	"time"
)

// swagger:model
type healthCheckData struct {
	// example: 25h53m33.540493171s
	Uptime    string    `json:"uptime"`

	// example: 10449
	Process   int       `json:"process"`
	Version   string    `json:"version"`
	BuildInfo buildInfo `json:"buildInfo"`
}

type buildInfo struct {
	// example: unknown
	Commit      string `json:"commit"`

	// example: unknown
	Branch      string `json:"branch"`

	// example: dev-build
	BuildNumber string `json:"buildNumber"`
}

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
// ---
// summary: Returns information about client
// description: Returns health check information about client
// responses:
//   200:
//     description: Health check information
//     schema:
//       "$ref": "#/definitions/healthCheckData"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/errorMessage"
func (hce *healthCheckEndpoint) HealthCheck(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	status := healthCheckData{
		Uptime:  hce.currentTimeFunc().Sub(hce.startTime).String(),
		Process: hce.processNumber,
		Version: metadata.VersionAsString(),
		BuildInfo: buildInfo{
			metadata.BuildCommit,
			metadata.BuildBranch,
			metadata.BuildNumber,
		},
	}
	utils.WriteAsJSON(status, writer)
}
