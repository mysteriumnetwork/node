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
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/go-rest/apierror"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/services"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/rs/zerolog/log"
)

// ServiceEndpoint struct represents management of service resource and it's sub-resources
type ServiceEndpoint struct {
	serviceManager     ServiceManager
	optionsParser      map[string]services.ServiceOptionsParser
	proposalRepository proposalRepository
	tequilaApiClient   *tequilapi_client.Client
}

var (
	// serviceTypeInvalid represents service type which is unknown to node
	serviceTypeInvalid = "<unknown>"
	// serviceOptionsInvalid represents service options which is unknown to node (i.e. invalid structure for given type)
	serviceOptionsInvalid struct{}
)

// NewServiceEndpoint creates and returns service endpoint
func NewServiceEndpoint(serviceManager ServiceManager, optionsParser map[string]services.ServiceOptionsParser, proposalRepository proposalRepository, tequilaApiClient *tequilapi_client.Client) *ServiceEndpoint {
	return &ServiceEndpoint{
		serviceManager:     serviceManager,
		optionsParser:      optionsParser,
		proposalRepository: proposalRepository,
		tequilaApiClient:   tequilaApiClient,
	}
}

// ServiceList provides a list of running services on the node.
// swagger:operation GET /services Service ServiceListResponse
//
//	---
//	summary: List of services
//	description: ServiceList provides a list of running services on the node.
//	responses:
//	  200:
//	    description: List of running services
//	    schema:
//	      "$ref": "#/definitions/ServiceListResponse"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (se *ServiceEndpoint) ServiceList(c *gin.Context) {
	includeAll := false
	includeAllStr := c.Request.URL.Query().Get("include_all")
	if len(includeAllStr) > 0 {
		var err error
		includeAll, err = strconv.ParseBool(includeAllStr)
		if err != nil {
			c.Error(apierror.BadRequestField(fmt.Sprintf("Failed to parse request: %s", err.Error()), "include_all", contract.ErrCodeServiceList))
			return
		}
	}

	instances := se.serviceManager.List(includeAll)

	statusResponse, err := se.toServiceListResponse(instances)
	if err != nil {
		c.Error(apierror.Internal("Cannot list services: "+err.Error(), contract.ErrCodeServiceList))
		return
	}
	utils.WriteAsJSON(statusResponse, c.Writer)
}

// ServiceGet provides info for requested service on the node.
// swagger:operation GET /services/:id Service serviceGet
//
//	---
//	summary: Information about service
//	description: ServiceGet provides info for requested service on the node.
//	responses:
//	  200:
//	    description: Service detailed information
//	    schema:
//	      "$ref": "#/definitions/ServiceInfoDTO"
//	  404:
//	    description: Service not found
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (se *ServiceEndpoint) ServiceGet(c *gin.Context) {
	id := service.ID(c.Param("id"))
	instance := se.serviceManager.Service(id)
	if instance == nil {
		c.Error(apierror.NotFound("Requested service not found"))
		return
	}

	statusResponse, err := se.toServiceInfoResponse(id, instance)
	if err != nil {
		c.Error(apierror.Internal("Cannot generate response: "+err.Error(), contract.ErrCodeServiceGet))
		return
	}
	utils.WriteAsJSON(statusResponse, c.Writer)
}

// ServiceStart starts requested service on the node.
// swagger:operation POST /services Service serviceStart
//
//	---
//	summary: Starts service
//	description: Provider starts serving new service to consumers
//	parameters:
//	  - in: body
//	    name: body
//	    description: Parameters in body (providerID) required for starting new service
//	    schema:
//	      $ref: "#/definitions/ServiceStartRequestDTO"
//	responses:
//	  201:
//	    description: Initiated service start
//	    schema:
//	      "$ref": "#/definitions/ServiceInfoDTO"
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  422:
//	    description: Unable to process the request at this point
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (se *ServiceEndpoint) ServiceStart(c *gin.Context) {
	sr, err := se.toServiceRequest(c.Request)
	if err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	if err := validateServiceRequest(sr); err != nil {
		c.Error(err)
		return
	}

	if se.isAlreadyRunning(sr) {
		c.Error(apierror.Unprocessable("Service already running", contract.ErrCodeServiceRunning))
		return
	}

	log.Info().Msgf("Service start options: %+v", sr)
	id, err := se.serviceManager.Start(
		identity.FromAddress(sr.ProviderID),
		sr.Type,
		sr.AccessPolicies.IDs,
		sr.Options,
	)
	if err == service.ErrorLocation {
		c.Error(apierror.Unprocessable("Cannot detect location", contract.ErrCodeServiceLocation))
		return
	} else if err != nil {
		c.Error(apierror.Internal("Cannot start service: "+err.Error(), contract.ErrCodeServiceStart))
		return
	}

	instance := se.serviceManager.Service(id)

	c.Status(http.StatusCreated)
	statusResponse, err := se.toServiceInfoResponse(id, instance)
	if err != nil {
		c.Error(apierror.Internal("Cannot generate response: "+err.Error(), contract.ErrCodeServiceGet))
		return
	}

	if ignoreUserConfig, _ := strconv.ParseBool(c.Query("ignore_user_config")); !ignoreUserConfig {
		se.updateActiveServicesInUserConfig()
	}

	utils.WriteAsJSON(statusResponse, c.Writer)
}

// ServiceStop stops service on the node.
// swagger:operation DELETE /services/:id Service serviceStop
//
//	---
//	summary: Stops service
//	description: Initiates service stop
//	responses:
//	  202:
//	    description: Service Stop initiated
//	  404:
//	    description: No service exists
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (se *ServiceEndpoint) ServiceStop(c *gin.Context) {
	id := service.ID(c.Param("id"))
	instance := se.serviceManager.Service(id)
	if instance == nil {
		c.Error(apierror.NotFound("Service not found"))
		return
	}

	if err := se.serviceManager.Stop(id); err != nil {
		c.Error(apierror.Internal("Cannot stop service: "+err.Error(), contract.ErrCodeServiceStop))
		return
	}

	if ignoreUserConfig, _ := strconv.ParseBool(c.Query("ignore_user_config")); !ignoreUserConfig {
		se.updateActiveServicesInUserConfig()
	}

	c.Status(http.StatusAccepted)
}

func (se *ServiceEndpoint) updateActiveServicesInUserConfig() {
	runningInstances := se.serviceManager.List(false)
	activeServices := make([]string, len(runningInstances))
	for i, service := range runningInstances {
		activeServices[i] = service.Type
	}
	config := map[string]interface{}{
		config.FlagActiveServices.Name: strings.Join(activeServices, ","),
	}
	se.tequilaApiClient.SetConfig(config)
}

func (se *ServiceEndpoint) isAlreadyRunning(sr contract.ServiceStartRequest) bool {
	for _, instance := range se.serviceManager.List(false) {
		if instance.ProviderID.Address == sr.ProviderID && instance.Type == sr.Type {
			return true
		}
	}
	return false
}

// AddRoutesForService adds service routes to given router
func AddRoutesForService(
	serviceManager ServiceManager,
	optionsParser map[string]services.ServiceOptionsParser,
	proposalRepository proposalRepository,
	tequilaApiClient *tequilapi_client.Client,
) func(*gin.Engine) error {
	serviceEndpoint := NewServiceEndpoint(serviceManager, optionsParser, proposalRepository, tequilaApiClient)

	return func(e *gin.Engine) error {
		g := e.Group("/services")
		{
			g.GET("", serviceEndpoint.ServiceList)
			g.POST("", serviceEndpoint.ServiceStart)
			g.GET("/:id", serviceEndpoint.ServiceGet)
			g.DELETE("/:id", serviceEndpoint.ServiceStop)
		}
		return nil
	}
}

func (se *ServiceEndpoint) toServiceRequest(req *http.Request) (contract.ServiceStartRequest, error) {
	var jsonData struct {
		ProviderID     string                          `json:"provider_id"`
		Type           string                          `json:"type"`
		Options        *json.RawMessage                `json:"options"`
		AccessPolicies *contract.ServiceAccessPolicies `json:"access_policies"`
	}
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&jsonData); err != nil {
		return contract.ServiceStartRequest{}, err
	}

	serviceOpts, _ := services.GetStartOptions(jsonData.Type)
	sr := contract.ServiceStartRequest{
		ProviderID: jsonData.ProviderID,
		Type:       se.toServiceType(jsonData.Type),
		Options:    se.toServiceOptions(jsonData.Type, jsonData.Options),
		AccessPolicies: &contract.ServiceAccessPolicies{
			IDs: serviceOpts.AccessPolicyList,
		},
	}
	if jsonData.AccessPolicies != nil {
		sr.AccessPolicies = jsonData.AccessPolicies
	}
	return sr, nil
}

func (se *ServiceEndpoint) toServiceType(value string) string {
	if value == "" {
		return ""
	}

	_, ok := se.optionsParser[value]
	if !ok {
		return serviceTypeInvalid
	}

	return value
}

func (se *ServiceEndpoint) toServiceOptions(serviceType string, value *json.RawMessage) service.Options {
	optionsParser, ok := se.optionsParser[serviceType]
	if !ok {
		return nil
	}

	options, err := optionsParser(value)
	if err != nil {
		return serviceOptionsInvalid
	}

	return options
}

func (se *ServiceEndpoint) toServiceInfoResponse(id service.ID, instance *service.Instance) (contract.ServiceInfoDTO, error) {
	priced, err := se.proposalRepository.EnrichProposalWithPrice(instance.Proposal)
	if err != nil {
		return contract.ServiceInfoDTO{}, err
	}

	var prop *contract.ProposalDTO
	if len(id) > 0 {
		tmp := contract.NewProposalDTO(priced)
		prop = &tmp
	}

	return contract.ServiceInfoDTO{
		ID:         string(id),
		ProviderID: instance.ProviderID.Address,
		Type:       instance.Type,
		Options:    instance.Options,
		Status:     string(instance.State()),
		Proposal:   prop,
	}, nil
}

func (se *ServiceEndpoint) toServiceListResponse(instances []*service.Instance) (contract.ServiceListResponse, error) {
	res := make([]contract.ServiceInfoDTO, 0)
	for _, instance := range instances {
		mapped, err := se.toServiceInfoResponse(instance.ID, instance)
		if err != nil {
			return nil, err
		}
		res = append(res, mapped)
	}
	return res, nil
}

func validateServiceRequest(sr contract.ServiceStartRequest) *apierror.APIError {
	v := apierror.NewValidator()
	if len(sr.ProviderID) == 0 {
		v.Required("provider_id")
	}
	if sr.Type == "" {
		v.Required("type")
	} else if sr.Type == serviceTypeInvalid {
		v.Invalid("type", "Invalid service type")
	}
	if sr.Options == serviceOptionsInvalid {
		v.Invalid("options", "Invalid options")
	}
	return v.Err()
}

// ServiceManager represents service manager that is used for services management.
type ServiceManager interface {
	Start(providerID identity.Identity, serviceType string, policies []string, options service.Options) (service.ID, error)
	Stop(id service.ID) error
	Service(id service.ID) *service.Instance
	Kill() error
	List(includeAll bool) []*service.Instance
}
