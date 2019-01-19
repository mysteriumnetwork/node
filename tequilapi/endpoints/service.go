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
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// swagger:model ServiceStatusDTO
type serviceStatusResponse struct {
	// example: Running
	Status string `json:"status"`
}

// ServiceEndpoint struct represents /service resource and it's sub-resources
type ServiceEndpoint struct {
	serviceManager ServiceManager
}

// NewServiceEndpoint creates and returns service endpoint
func NewServiceEndpoint(serviceManager ServiceManager) *ServiceEndpoint {
	return &ServiceEndpoint{
		serviceManager: serviceManager,
	}
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
		Status: string(service.NotRunning),
	}
	utils.WriteAsJSON(statusResponse, resp)
}

// AddRoutesForService adds service routes to given router
func AddRoutesForService(router *httprouter.Router, serviceManager *service.Manager) {
	serviceEndpoint := NewServiceEndpoint(serviceManager)
	router.GET("/service", serviceEndpoint.Status)
}

type ServiceManager interface {
	Start(providerID identity.Identity, serviceType string, options service.Options) (err error)
	Kill() error
}
