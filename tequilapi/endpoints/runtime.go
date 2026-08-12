/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
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
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/mysteriumnetwork/node/core/service"
	runtime_service "github.com/mysteriumnetwork/node/services/runtime"
	"github.com/mysteriumnetwork/node/services/runtime/registry"
	runtime_service_impl "github.com/mysteriumnetwork/node/services/runtime/service"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// RuntimeEndpoint manages runtime workload definitions: the install and delete
// half of the runtime service lifecycle that POST/DELETE /services cannot
// reach, because those act on service instances of an already installed
// definition.
//
// It deliberately imposes the same preconditions as the NATS control plane, the
// other surface that manages these definitions. A definition installed here
// must be one the control plane would also have accepted, or the two surfaces
// would disagree about what state a node is allowed to be in.
type RuntimeEndpoint struct {
	backend runtime_service_impl.Backend
	// installer is the only path to an installed workload; the backend's own
	// Create cannot be reached without the approval this produces.
	installer      runtime_service_impl.Installer
	serviceManager ServiceManager
}

// NewRuntimeEndpoint creates and returns the runtime endpoint.
func NewRuntimeEndpoint(
	backend runtime_service_impl.Backend,
	installer runtime_service_impl.Installer,
	serviceManager ServiceManager,
) *RuntimeEndpoint {
	return &RuntimeEndpoint{
		backend:        backend,
		installer:      installer,
		serviceManager: serviceManager,
	}
}

// RuntimeStatus reports whether this host can run workloads.
// swagger:operation GET /runtime/status Runtime runtimeStatus
//
//	---
//	summary: Runtime availability
//	description: Reports whether workloads can run on this host, at which isolation level, and what is blocking them when they cannot.
//	responses:
//	  200:
//	    description: Runtime status
//	    schema:
//	      "$ref": "#/definitions/RuntimeStatusDTO"
//	  503:
//	    description: Node was built or configured without runtime support
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (re *RuntimeEndpoint) RuntimeStatus(c *gin.Context) {
	if err := re.requireBackend(); err != nil {
		c.Error(err)
		return
	}

	capabilities, detailed := re.backend.Capabilities()
	availability := re.backend.Availability()

	utils.WriteAsJSON(contract.RuntimeStatusDTO{
		Available:            availability.Available,
		Reason:               availability.Reason,
		Runtime:              re.backend.Status(),
		Capabilities:         capabilities,
		DetailedCapabilities: detailed,
	}, c.Writer)
}

// RuntimeServiceList lists installed runtime workload definitions.
// swagger:operation GET /runtime/services Runtime runtimeServiceList
//
//	---
//	summary: List of installed runtime services
//	description: Lists every installed runtime workload definition, running or not. GET /services only reports running instances.
//	responses:
//	  200:
//	    description: List of installed runtime services
//	    schema:
//	      "$ref": "#/definitions/RuntimeServiceListResponse"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  503:
//	    description: Node was built or configured without runtime support
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (re *RuntimeEndpoint) RuntimeServiceList(c *gin.Context) {
	if err := re.requireBackend(); err != nil {
		c.Error(err)
		return
	}

	installed, err := re.backend.List()
	if err != nil {
		c.Error(apierror.Internal("Cannot list runtime services: "+err.Error(), contract.ErrCodeRuntimeList))
		return
	}

	response := contract.RuntimeServiceListResponse{Services: make([]contract.RuntimeServiceInfoDTO, 0, len(installed))}
	for _, info := range installed {
		response.Services = append(response.Services, re.toRuntimeServiceInfoDTO(info))
	}
	utils.WriteAsJSON(response, c.Writer)
}

// RuntimeServiceGet returns a single installed runtime workload definition.
// swagger:operation GET /runtime/services/{name} Runtime runtimeServiceGet
//
//	---
//	summary: Installed runtime service
//	description: Returns one installed runtime workload definition. The name may be given with or without the runtime- prefix.
//	parameters:
//	  - in: path
//	    name: name
//	    description: Runtime service name, with or without the runtime- prefix
//	    type: string
//	    required: true
//	responses:
//	  200:
//	    description: Installed runtime service
//	    schema:
//	      "$ref": "#/definitions/RuntimeServiceInfoDTO"
//	  400:
//	    description: Invalid service name
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  404:
//	    description: No such runtime service is installed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  503:
//	    description: Node was built or configured without runtime support
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (re *RuntimeEndpoint) RuntimeServiceGet(c *gin.Context) {
	if err := re.requireBackend(); err != nil {
		c.Error(err)
		return
	}

	serviceType, apiErr := normalizeRuntimeServiceName(c.Param("name"))
	if apiErr != nil {
		c.Error(apiErr)
		return
	}

	info, exists, err := re.backend.Get(serviceType)
	if err != nil {
		c.Error(apierror.Internal("Cannot get runtime service: "+err.Error(), contract.ErrCodeRuntimeGet))
		return
	}
	if !exists {
		c.Error(apierror.NotFound("Requested runtime service is not installed"))
		return
	}

	utils.WriteAsJSON(re.toRuntimeServiceInfoDTO(info), c.Writer)
}

// RuntimeServiceInstall installs a runtime workload definition.
// swagger:operation POST /runtime/services Runtime runtimeServiceInstall
//
//	---
//	summary: Installs a runtime service
//	description: >
//	  Installs the workload the service registry publishes under the given name. The artifact cannot be chosen by the
//	  caller: it is resolved and digest-pinned by the registry, so a name the registry does not list is refused.
//	  Installing does not start the service; start it through POST /services with the returned name as the type.
//	parameters:
//	  - in: body
//	    name: body
//	    description: Name of the registry entry to install
//	    schema:
//	      $ref: "#/definitions/RuntimeServiceInstallRequestDTO"
//	responses:
//	  201:
//	    description: Runtime service installed
//	    schema:
//	      "$ref": "#/definitions/RuntimeServiceInfoDTO"
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  422:
//	    description: Runtime service cannot be installed in the node's current state
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  503:
//	    description: Node was built or configured without runtime support
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (re *RuntimeEndpoint) RuntimeServiceInstall(c *gin.Context) {
	if err := re.requireBackend(); err != nil {
		c.Error(err)
		return
	}
	if re.installer == nil {
		c.Error(apierror.Error(
			http.StatusServiceUnavailable,
			"Runtime service registry is not configured; runtime services cannot be installed",
			contract.ErrCodeRuntimeNotConfigured,
		))
		return
	}

	var request contract.RuntimeServiceInstallRequest
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	serviceType, apiErr := normalizeRuntimeServiceName(request.Name)
	if apiErr != nil {
		c.Error(apiErr)
		return
	}

	// An install publishes a service the node then has to be able to run, so
	// the same preconditions the control plane enforces for a create apply.
	if err := re.requireRuntimeAvailable(); err != nil {
		c.Error(err)
		return
	}
	if err := re.requireRuntimeServiceActive(); err != nil {
		c.Error(err)
		return
	}

	log.Info().Str("service", serviceType).Msg("Installing runtime service")
	if err := re.installer.Install(runtime_service_impl.CreateOptions{
		Name:                serviceType,
		MinimumRuntimeLevel: request.MinimumRuntimeLevel,
	}); err != nil {
		if errors.Is(err, registry.ErrNotListed) {
			c.Error(apierror.Unprocessable(
				"Cannot install runtime service: "+err.Error(),
				contract.ErrCodeRuntimeNotListed,
			))
			return
		}
		if errors.Is(err, runtime_service_impl.ErrNotConfigured) {
			// Nothing about the request would make this succeed.
			c.Error(apierror.Error(
				http.StatusServiceUnavailable,
				"Cannot install runtime service: "+err.Error(),
				contract.ErrCodeRuntimeNotConfigured,
			))
			return
		}
		c.Error(apierror.Unprocessable(
			"Cannot install runtime service: "+err.Error(),
			contract.ErrCodeRuntimeInstall,
		))
		return
	}

	info, exists, err := re.backend.Get(serviceType)
	if err != nil || !exists {
		// Install verified the definition before returning, so this is a
		// response-building failure rather than a failed install.
		c.Error(apierror.Internal("Cannot read installed runtime service", contract.ErrCodeRuntimeGet))
		return
	}

	c.Status(http.StatusCreated)
	utils.WriteAsJSON(re.toRuntimeServiceInfoDTO(info), c.Writer)
}

// RuntimeServiceDelete removes a runtime workload definition.
// swagger:operation DELETE /runtime/services/{name} Runtime runtimeServiceDelete
//
//	---
//	summary: Deletes a runtime service
//	description: >
//	  Stops the service if it is running and removes its definition. Deleting is not the same as stopping: a stopped
//	  service can be started again, a deleted one has to be installed again first.
//	parameters:
//	  - in: path
//	    name: name
//	    description: Runtime service name, with or without the runtime- prefix
//	    type: string
//	    required: true
//	responses:
//	  202:
//	    description: Runtime service deletion initiated
//	  400:
//	    description: Invalid service name
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  404:
//	    description: No such runtime service is installed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  422:
//	    description: Runtime service cannot be deleted in the node's current state
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  503:
//	    description: Node was built or configured without runtime support
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (re *RuntimeEndpoint) RuntimeServiceDelete(c *gin.Context) {
	if err := re.requireBackend(); err != nil {
		c.Error(err)
		return
	}

	serviceType, apiErr := normalizeRuntimeServiceName(c.Param("name"))
	if apiErr != nil {
		c.Error(apiErr)
		return
	}
	if err := re.requireRuntimeServiceActive(); err != nil {
		c.Error(err)
		return
	}

	_, exists, err := re.backend.Get(serviceType)
	if err != nil {
		c.Error(apierror.Internal("Cannot get runtime service: "+err.Error(), contract.ErrCodeRuntimeGet))
		return
	}
	if !exists {
		c.Error(apierror.NotFound("Requested runtime service is not installed"))
		return
	}

	// The node-side service goes down first: a definition removed underneath a
	// running instance would leave a published proposal for a workload that no
	// longer exists.
	for _, instance := range re.runningInstances(serviceType) {
		if err := re.serviceManager.Stop(instance); err != nil {
			c.Error(apierror.Internal("Cannot stop runtime service: "+err.Error(), contract.ErrCodeServiceStop))
			return
		}
	}

	log.Info().Str("service", serviceType).Msg("Deleting runtime service")
	if err := re.backend.Delete(serviceType); err != nil {
		c.Error(apierror.Internal("Cannot delete runtime service: "+err.Error(), contract.ErrCodeRuntimeDelete))
		return
	}

	c.Status(http.StatusAccepted)
}

// requireBackend refuses every request on a node that has no runtime backend at
// all, which is a build or platform property rather than a bad request.
func (re *RuntimeEndpoint) requireBackend() *apierror.APIError {
	if re.backend != nil {
		return nil
	}
	return apierror.Error(
		http.StatusServiceUnavailable,
		"Runtime services are not supported on this node",
		contract.ErrCodeRuntimeNotConfigured,
	)
}

// requireRuntimeAvailable refuses an install the host cannot honour, so a node
// does not end up publishing a service it can never start.
func (re *RuntimeEndpoint) requireRuntimeAvailable() *apierror.APIError {
	reason, unavailable := runtime_service_impl.UnavailableReason(re.backend)
	if !unavailable {
		return nil
	}
	return apierror.Unprocessable("Runtime is unavailable: "+reason, contract.ErrCodeRuntimeUnavailable)
}

// requireRuntimeServiceActive mirrors the control plane's rule that runtime-*
// services are managed only while the parent runtime service is running. The
// parent service is what reconciles workloads and fronts them on the network,
// so a definition installed without it would be inert and invisible.
func (re *RuntimeEndpoint) requireRuntimeServiceActive() *apierror.APIError {
	for _, instance := range re.serviceManager.List(false) {
		if instance.Type == runtime_service.ServiceType {
			return nil
		}
	}
	return apierror.Unprocessable(
		"Runtime service must be running to manage runtime services",
		contract.ErrCodeRuntimeInactive,
	)
}

func (re *RuntimeEndpoint) runningInstances(serviceType string) []service.ID {
	var ids []service.ID
	for _, instance := range re.serviceManager.List(false) {
		if instance.Type == serviceType {
			ids = append(ids, instance.ID)
		}
	}
	return ids
}

func (re *RuntimeEndpoint) toRuntimeServiceInfoDTO(info runtime_service_impl.ServiceInfo) contract.RuntimeServiceInfoDTO {
	dto := contract.RuntimeServiceInfoDTO{
		Name:         info.Name,
		State:        info.State,
		DesiredState: info.Desired,
		Options:      info.Options,
	}
	if ids := re.runningInstances(info.Name); len(ids) > 0 {
		dto.ServiceID = string(ids[0])
	}
	return dto
}

// normalizeRuntimeServiceName resolves a caller-supplied name onto the
// canonical runtime-<name> service type, the same way the control plane and the
// service registry do, so a service is reachable by the name an operator reads
// in the registry.
func normalizeRuntimeServiceName(name string) (string, *apierror.APIError) {
	serviceType := runtime_service.NormalizeServiceType(name)
	if serviceType == "" {
		v := apierror.NewValidator()
		v.Required("name")
		return "", v.Err()
	}
	return serviceType, nil
}

// AddRoutesForRuntime adds runtime service management routes to given router.
func AddRoutesForRuntime(
	backend runtime_service_impl.Backend,
	installer runtime_service_impl.Installer,
	serviceManager ServiceManager,
) func(*gin.Engine) error {
	runtimeEndpoint := NewRuntimeEndpoint(backend, installer, serviceManager)

	return func(e *gin.Engine) error {
		g := e.Group("/runtime")
		{
			g.GET("/status", runtimeEndpoint.RuntimeStatus)
			g.GET("/services", runtimeEndpoint.RuntimeServiceList)
			g.POST("/services", runtimeEndpoint.RuntimeServiceInstall)
			g.GET("/services/:name", runtimeEndpoint.RuntimeServiceGet)
			g.DELETE("/services/:name", runtimeEndpoint.RuntimeServiceDelete)
		}
		return nil
	}
}
