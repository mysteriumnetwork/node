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
	"fmt"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

const (
	// statusConnectCancelled indicates that connect request was cancelled by user. Since there is no such concept in REST
	// operations, custom client error code is defined. Maybe in later times a better idea will come how to handle these situations
	statusConnectCancelled = 499
)

// ProposalGetter defines interface to fetch currently active service proposal by id
type ProposalGetter interface {
	GetProposal(id market.ProposalID) (*market.ServiceProposal, error)
}

type identityRegistry interface {
	GetRegistrationStatus(int64, identity.Identity) (registry.RegistrationStatus, error)
}

// ConnectionEndpoint struct represents /connection resource and it's subresources
type ConnectionEndpoint struct {
	manager       connection.MultiManager
	publisher     eventbus.Publisher
	stateProvider stateProvider
	// TODO connection should use concrete proposal from connection params and avoid going to marketplace
	proposalRepository proposalRepository
	identityRegistry   identityRegistry
	addressProvider    addressProvider
}

// NewConnectionEndpoint creates and returns connection endpoint
func NewConnectionEndpoint(manager connection.MultiManager, stateProvider stateProvider, proposalRepository proposalRepository, identityRegistry identityRegistry, publisher eventbus.Publisher, addressProvider addressProvider) *ConnectionEndpoint {
	return &ConnectionEndpoint{
		manager:            manager,
		publisher:          publisher,
		stateProvider:      stateProvider,
		proposalRepository: proposalRepository,
		identityRegistry:   identityRegistry,
		addressProvider:    addressProvider,
	}
}

// Status returns status of connection
// swagger:operation GET /connection Connection connectionStatus
//
//	---
//	summary: Returns connection status
//	description: Returns status of current connection
//	responses:
//	  200:
//	    description: Status
//	    schema:
//	      "$ref": "#/definitions/ConnectionInfoDTO"
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (ce *ConnectionEndpoint) Status(c *gin.Context) {
	n := 0
	id := c.Query("id")
	if len(id) > 0 {
		var err error
		n, err = strconv.Atoi(id)
		if err != nil {
			c.Error(apierror.ParseFailed())
			return
		}
	}
	status := ce.manager.Status(n)
	statusResponse := contract.NewConnectionInfoDTO(status)
	utils.WriteAsJSON(statusResponse, c.Writer)
}

// Create starts new connection
// swagger:operation PUT /connection Connection connectionCreate
//
//	---
//	summary: Starts new connection
//	description: Consumer opens connection to provider
//	parameters:
//	  - in: body
//	    name: body
//	    description: Parameters in body (consumer_id, provider_id, service_type) required for creating new connection
//	    schema:
//	      $ref: "#/definitions/ConnectionCreateRequestDTO"
//	responses:
//	  201:
//	    description: Connection started
//	    schema:
//	      "$ref": "#/definitions/ConnectionInfoDTO"
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
func (ce *ConnectionEndpoint) Create(c *gin.Context) {
	hermes, err := ce.addressProvider.GetActiveHermes(config.GetInt64(config.FlagChainID))
	if err != nil {
		c.Error(apierror.Internal("Failed to get active hermes", contract.ErrCodeActiveHermes))
		return
	}

	cr, err := toConnectionRequest(c.Request, hermes.Hex())
	if err != nil {
		ce.publisher.Publish(quality.AppTopicConnectionEvents, (&contract.ConnectionCreateRequest{}).Event(quality.StagePraseRequest, err.Error()))
		c.Error(apierror.ParseFailed())
		return
	}

	if err := cr.Validate(); err != nil {
		ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageValidateRequest, err.Detail()))
		c.Error(err)
		return
	}

	consumerID := identity.FromAddress(cr.ConsumerID)
	status, err := ce.identityRegistry.GetRegistrationStatus(config.GetInt64(config.FlagChainID), consumerID)
	if err != nil {
		ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageRegistrationGetStatus, err.Error()))
		log.Error().Err(err).Stack().Msg("Could not check registration status")
		c.Error(apierror.Internal("Failed to check ID registration status: "+err.Error(), contract.ErrCodeIDRegistrationCheck))
		return
	}

	switch status {
	case registry.Unregistered, registry.RegistrationError, registry.Unknown:
		ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageRegistrationUnregistered, ""))
		log.Error().Msgf("Identity %q is not registered, aborting...", cr.ConsumerID)
		c.Error(apierror.Unprocessable(fmt.Sprintf("Identity %q is not registered. Please register the identity first", cr.ConsumerID), contract.ErrCodeIDNotRegistered))
		return
	case registry.InProgress:
		ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageRegistrationInProgress, ""))
		log.Info().Msgf("identity %q registration is in progress, continuing...", cr.ConsumerID)
	case registry.Registered:
		ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageRegistrationRegistered, ""))
		log.Info().Msgf("identity %q is registered, continuing...", cr.ConsumerID)
	default:
		ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageRegistrationUnknown, ""))
		log.Error().Msgf("identity %q has unknown status, aborting...", cr.ConsumerID)
		c.Error(apierror.Unprocessable(fmt.Sprintf("Identity %q has unknown status. Aborting", cr.ConsumerID), contract.ErrCodeIDStatusUnknown))
		return
	}

	if len(cr.ProviderID) > 0 {
		cr.Filter.Providers = append(cr.Filter.Providers, cr.ProviderID)
	}

	f := &proposal.Filter{
		ServiceType:             cr.ServiceType,
		LocationCountry:         cr.Filter.CountryCode,
		ProviderIDs:             cr.Filter.Providers,
		IPType:                  cr.Filter.IPType,
		IncludeMonitoringFailed: cr.Filter.IncludeMonitoringFailed,
		AccessPolicy:            "all",
	}
	proposalLookup := connection.FilteredProposals(f, cr.Filter.SortBy, ce.proposalRepository)

	err = ce.manager.Connect(consumerID, common.HexToAddress(cr.HermesID), proposalLookup, getConnectOptions(cr))
	if err != nil {
		switch err {
		case connection.ErrAlreadyExists:
			ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageConnectionAlreadyExists, err.Error()))
			c.Error(apierror.Unprocessable("Connection already exists", contract.ErrCodeConnectionAlreadyExists))
		case connection.ErrConnectionCancelled:
			ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageConnectionCanceled, err.Error()))
			c.Error(apierror.Unprocessable("Connection cancelled", contract.ErrCodeConnectionCancelled))
		default:
			ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageConnectionUnknownError, err.Error()))
			log.Error().Err(err).Msg("Failed to connect")
			c.Error(apierror.Internal("Failed to connect: "+err.Error(), contract.ErrCodeConnect))
		}
		return
	}

	ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageConnectionOK, ""))
	c.Status(http.StatusCreated)

	statusResp := ce.manager.Status(cr.ConnectOptions.ProxyPort)
	statusResponse := contract.NewConnectionInfoDTO(statusResp)
	utils.WriteAsJSON(statusResponse, c.Writer)
}

// Kill stops connection
// swagger:operation DELETE /connection Connection connectionCancel
//
//	---
//	summary: Stops connection
//	description: Stops current connection
//	responses:
//	  202:
//	    description: Connection stopped
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  422:
//	    description: Unable to process the request at this point (e.g. no active connection exists)
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (ce *ConnectionEndpoint) Kill(c *gin.Context) {
	n := 0
	id := c.Query("id")
	if len(id) > 0 {
		var err error
		n, err = strconv.Atoi(id)
		if err != nil {
			c.Error(apierror.ParseFailed())
			return
		}
	}

	err := ce.manager.Disconnect(n)
	if err != nil {
		switch err {
		case connection.ErrNoConnection:
			c.Error(apierror.Unprocessable("No connection exists", contract.ErrCodeNoConnectionExists))
		default:
			c.Error(apierror.Internal("Could not disconnect: "+err.Error(), contract.ErrCodeDisconnect))
		}
		return
	}
	c.Status(http.StatusAccepted)
}

// GetStatistics returns statistics about current connection
// swagger:operation GET /connection/statistics Connection connectionStatistics
//
//	---
//	summary: Returns connection statistics
//	description: Returns statistics about current connection
//	responses:
//	  200:
//	    description: Connection statistics
//	    schema:
//	      "$ref": "#/definitions/ConnectionStatisticsDTO"
func (ce *ConnectionEndpoint) GetStatistics(c *gin.Context) {
	id := c.Query("id")
	conn := ce.stateProvider.GetConnection(id)

	response := contract.NewConnectionStatisticsDTO(conn.Session, conn.Statistics, conn.Throughput, conn.Invoice)
	utils.WriteAsJSON(response, c.Writer)
}

// GetTraffic returns traffic information about requested connection
// swagger:operation GET /connection/traffic Connection connectionTraffic
//
//	---
//	summary: Returns connection traffic information
//	description: Returns traffic information about requested connection
//	responses:
//	  200:
//	    description: Connection traffic
//	    schema:
//	      "$ref": "#/definitions/ConnectionTrafficDTO"
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (ce *ConnectionEndpoint) GetTraffic(c *gin.Context) {
	n := 0
	id := c.Query("id")
	if len(id) > 0 {
		var err error
		n, err = strconv.Atoi(id)
		if err != nil {
			c.Error(apierror.ParseFailed())
			return
		}
	}

	traffic := ce.manager.Stats(n)

	response := contract.ConnectionTrafficDTO{
		BytesSent:     traffic.BytesSent,
		BytesReceived: traffic.BytesReceived,
	}
	utils.WriteAsJSON(response, c.Writer)
}

type proposalRepository interface {
	Proposal(id market.ProposalID) (*proposal.PricedServiceProposal, error)
	Proposals(filter *proposal.Filter) ([]proposal.PricedServiceProposal, error)
	Countries(filter *proposal.Filter) (map[string]int, error)
	EnrichProposalWithPrice(in market.ServiceProposal) (proposal.PricedServiceProposal, error)
}

// AddRoutesForConnection adds connections routes to given router
func AddRoutesForConnection(
	manager connection.MultiManager,
	stateProvider stateProvider,
	proposalRepository proposalRepository,
	identityRegistry identityRegistry,
	publisher eventbus.Publisher,
	addressProvider addressProvider,
) func(*gin.Engine) error {
	connectionEndpoint := NewConnectionEndpoint(manager, stateProvider, proposalRepository, identityRegistry, publisher, addressProvider)
	return func(e *gin.Engine) error {
		connGroup := e.Group("")
		{
			connGroup.GET("/connection", connectionEndpoint.Status)
			connGroup.PUT("/connection", connectionEndpoint.Create)
			connGroup.DELETE("/connection", connectionEndpoint.Kill)
			connGroup.GET("/connection/statistics", connectionEndpoint.GetStatistics)
			connGroup.GET("/connection/traffic", connectionEndpoint.GetTraffic)
		}
		return nil
	}
}

func toConnectionRequest(req *http.Request, defaultHermes string) (*contract.ConnectionCreateRequest, error) {
	connectionRequest := contract.ConnectionCreateRequest{
		ConnectOptions: contract.ConnectOptions{
			DisableKillSwitch: false,
			DNS:               connection.DNSOptionAuto,
		},
		HermesID: defaultHermes,
	}
	err := json.NewDecoder(req.Body).Decode(&connectionRequest)
	if err != nil {
		return nil, err
	}
	return &connectionRequest, nil
}

func getConnectOptions(cr *contract.ConnectionCreateRequest) connection.ConnectParams {
	dns := connection.DNSOptionAuto
	if cr.ConnectOptions.DNS != "" {
		dns = cr.ConnectOptions.DNS
	}

	return connection.ConnectParams{
		DisableKillSwitch: cr.ConnectOptions.DisableKillSwitch,
		DNS:               dns,
		ProxyPort:         cr.ConnectOptions.ProxyPort,
	}
}
