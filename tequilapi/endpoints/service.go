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
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// swagger:model ServiceStatusDTO
type serviceStatusResponse struct {
	// example: Running
	Status string `json:"status"`
}

// ServiceEndpoint struct represents /service resource and it's sub-resources
type ServiceEndpoint struct{}

// NewServiceEndpoint creates and returns service endpoint
func NewServiceEndpoint() *ServiceEndpoint {
	return &ServiceEndpoint{}
}

// Status returns status of service
// swagger:operation GET /service Service serviceStatus
// ---
// summary: Returns service status
// description: Returns status of current service
// responses:
//   200:
//     description: Status
//     schema:
//       "$ref": "#/definitions/ServiceStatusDTO"
func (se *ServiceEndpoint) Status(resp http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	statusResponse := serviceStatusResponse{
		// TODO Create constants for service running states
		Status: "",
	}
	utils.WriteAsJSON(statusResponse, resp)
}

// AddRoutesForService adds service routes to given router
func AddRoutesForService(router *httprouter.Router) {
	serviceEndpoint := NewServiceEndpoint()
	router.GET("/service", serviceEndpoint.Status)
}
