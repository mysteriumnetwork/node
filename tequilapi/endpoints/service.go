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
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/mysteriumnetwork/node/tequilapi/validation"
)

// swagger:model ServiceRequestDTO
type serviceRequest struct {
	// provider identity
	// required: true
	// example: 0x0000000000000000000000000000000000000002
	ProviderID string `json:"providerId"`

	// service type. Possible values are "openvpn" and "noop"
	// required: false
	// default: openvpn
	// example: openvpn
	ServiceType string `json:"serviceType"`
}

// swagger:model ServiceStatusDTO
type serviceResponse struct {
	// example: Running
	Status string `json:"status"`
}

// ServiceEndpoint struct represents /service resource and it's sub-resources
type ServiceEndpoint struct {
	serviceManager ServiceManager
	serviceState   service.State
}

// NewServiceEndpoint creates and returns service endpoint
func NewServiceEndpoint(serviceManager ServiceManager) *ServiceEndpoint {
	return &ServiceEndpoint{
		serviceManager: serviceManager,
		serviceState:   service.NotRunning,
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
	statusResponse := toServiceResponse(se.serviceState)
	utils.WriteAsJSON(statusResponse, resp)
}

// Create starts new service
// swagger:operation PUT /service Service createService
// ---
// summary: Starts starts service
// description: Provider starts serving new service to consumers
// parameters:
//   - in: body
//     name: body
//     description: Parameters in body (providerId) required for starting new service
//     schema:
//       $ref: "#/definitions/ServiceRequestDTO"
// responses:
//   201:
//     description: Service started
//     schema:
//       "$ref": "#/definitions/ServiceStatusDTO"
//   400:
//     description: Bad request
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   409:
//     description: Conflict. Service is already running
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   422:
//     description: Parameters validation error
//     schema:
//       "$ref": "#/definitions/ValidationErrorDTO"
//   499:
//     description: Service was cancelled
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ce *ServiceEndpoint) Create(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	sr, err := toServiceRequest(req)
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}

	errorMap := validateServiceRequest(sr)
	if errorMap.HasErrors() {
		utils.SendValidationErrorMessage(resp, errorMap)
		return
	}

	ce.serviceState = service.Running

	resp.WriteHeader(http.StatusCreated)
	ce.Status(resp, req, params)
}

// AddRoutesForService adds service routes to given router
func AddRoutesForService(router *httprouter.Router, serviceManager *service.Manager) {
	serviceEndpoint := NewServiceEndpoint(serviceManager)
	router.GET("/service", serviceEndpoint.Status)
	router.PUT("/service", serviceEndpoint.Create)
}

func toServiceRequest(req *http.Request) (*serviceRequest, error) {
	var serviceRequest = serviceRequest{}
	err := json.NewDecoder(req.Body).Decode(&serviceRequest)

	return &serviceRequest, err
}

func toServiceResponse(state service.State) serviceResponse {
	return serviceResponse{
		Status: string(state),
	}
}

func validateServiceRequest(cr *serviceRequest) *validation.FieldErrorMap {
	errors := validation.NewErrorMap()
	if len(cr.ProviderID) == 0 {
		errors.ForField("providerId").AddError("required", "Field is required")
	}
	if cr.ServiceType == "" {
		errors.ForField("serviceType").AddError("required", "Field is required")
	}
	return errors
}

type ServiceManager interface {
	Start(providerID identity.Identity, serviceType string, options service.Options) (err error)
	Kill() error
}
