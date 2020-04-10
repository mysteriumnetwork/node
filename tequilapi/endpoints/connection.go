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

	"github.com/ethereum/go-ethereum/common"
	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/mysteriumnetwork/node/tequilapi/validation"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// statusConnectCancelled indicates that connect request was cancelled by user. Since there is no such concept in REST
// operations, custom client error code is defined. Maybe in later times a better idea will come how to handle these situations
const statusConnectCancelled = 499

// ConnectOptions holds tequilapi connect options
// swagger:model ConnectOptionsDTO
type ConnectOptions struct {
	// kill switch option restricting communication only through VPN
	// required: false
	// example: true
	DisableKillSwitch bool `json:"kill_switch"`
	// DNS to use
	// required: false
	// default: auto
	// example: auto, provider, system, "1.1.1.1,8.8.8.8"
	DNS connection.DNSOption `json:"dns"`
}

// swagger:model ConnectionRequestDTO
type connectionRequest struct {
	// consumer identity
	// required: true
	// example: 0x0000000000000000000000000000000000000001
	ConsumerID string `json:"consumer_id"`

	// provider identity
	// required: true
	// example: 0x0000000000000000000000000000000000000002
	ProviderID string `json:"provider_id"`

	// accountant identity
	// required: true
	// example: 0x0000000000000000000000000000000000000003
	AccountantID string `json:"accountant_id"`

	// service type. Possible values are "openvpn", "wireguard" and "noop"
	// required: false
	// default: openvpn
	// example: openvpn
	ServiceType string `json:"service_type"`

	// connect options
	// required: false
	ConnectOptions ConnectOptions `json:"connect_options,omitempty"`
}

// swagger:model ConnectionStatusDTO
type connectionResponse struct {
	// example: 0x00
	ConsumerID string `json:"consumer_id,omitempty"`

	// example: 0x00
	AccountantID string `json:"accountant_id,omitempty"`

	// example: Connected
	Status string `json:"status"`

	// example: 4cfb0324-daf6-4ad8-448b-e61fe0a1f918
	SessionID string `json:"session_id,omitempty"`

	// example: {"id":1,"provider_id":"0x71ccbdee7f6afe85a5bc7106323518518cd23b94","servcie_type":"openvpn","service_definition":{"location_originate":{"asn":"","country":"CA"}}}
	Proposal *proposalDTO `json:"proposal,omitempty"`
}

// swagger:model IPDTO
type ipResponse struct {
	// public IP address
	// example: 127.0.0.1
	IP string `json:"ip"`
}

// ProposalGetter defines interface to fetch currently active service proposal by id
type ProposalGetter interface {
	GetProposal(id market.ProposalID) (*market.ServiceProposal, error)
}

type identityRegistry interface {
	GetRegistrationStatus(identity.Identity) (registry.RegistrationStatus, error)
}

// ConnectionEndpoint struct represents /connection resource and it's subresources
type ConnectionEndpoint struct {
	manager       connection.Manager
	stateProvider stateProvider
	//TODO connection should use concrete proposal from connection params and avoid going to marketplace
	proposalRepository proposal.Repository
	identityRegistry   identityRegistry
}

// NewConnectionEndpoint creates and returns connection endpoint
func NewConnectionEndpoint(manager connection.Manager, stateProvider stateProvider, proposalRepository proposal.Repository, identityRegistry identityRegistry) *ConnectionEndpoint {
	return &ConnectionEndpoint{
		manager:            manager,
		stateProvider:      stateProvider,
		proposalRepository: proposalRepository,
		identityRegistry:   identityRegistry,
	}
}

// Status returns status of connection
// swagger:operation GET /connection Connection connectionStatus
// ---
// summary: Returns connection status
// description: Returns status of current connection
// responses:
//   200:
//     description: Status
//     schema:
//       "$ref": "#/definitions/ConnectionStatusDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ce *ConnectionEndpoint) Status(resp http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	statusResponse := toConnectionResponse(ce.manager.Status())
	utils.WriteAsJSON(statusResponse, resp)
}

// Create starts new connection
// swagger:operation PUT /connection Connection connectionCreate
// ---
// summary: Starts new connection
// description: Consumer opens connection to provider
// parameters:
//   - in: body
//     name: body
//     description: Parameters in body (consumer_id, provider_id, service_type) required for creating new connection
//     schema:
//       $ref: "#/definitions/ConnectionRequestDTO"
// responses:
//   201:
//     description: Connection started
//     schema:
//       "$ref": "#/definitions/ConnectionStatusDTO"
//   400:
//     description: Bad request
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   409:
//     description: Conflict. Connection already exists
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   422:
//     description: Parameters validation error
//     schema:
//       "$ref": "#/definitions/ValidationErrorDTO"
//   499:
//     description: Connection was cancelled
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ce *ConnectionEndpoint) Create(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	cr, err := toConnectionRequest(req)
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}

	status, err := ce.identityRegistry.GetRegistrationStatus(identity.FromAddress(cr.ConsumerID))
	if err != nil {
		log.Error().Err(err).Stack().Msg("could not check registration status")
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	switch status {
	case registry.Unregistered, registry.InProgress, registry.RegistrationError:
		log.Warn().Msgf("identity %q is not registered, aborting...", cr.ConsumerID)
		utils.SendError(resp, fmt.Errorf("identity %q is not registered. Please register the identity first", cr.ConsumerID), http.StatusExpectationFailed)
		return
	}

	log.Info().Msgf("identity %q is registered, continuing...", cr.ConsumerID)

	errorMap := validateConnectionRequest(cr)
	if errorMap.HasErrors() {
		utils.SendValidationErrorMessage(resp, errorMap)
		return
	}

	// TODO Pass proposal ID directly in request
	proposal, err := ce.proposalRepository.Proposal(market.ProposalID{
		ProviderID:  cr.ProviderID,
		ServiceType: cr.ServiceType,
	})
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}
	if proposal == nil {
		utils.SendError(resp, errors.New("provider has no service proposals"), http.StatusBadRequest)
		return
	}

	connectOptions := getConnectOptions(cr)
	err = ce.manager.Connect(identity.FromAddress(cr.ConsumerID), common.HexToAddress(cr.AccountantID), *proposal, connectOptions)

	if err != nil {
		switch err {
		case connection.ErrAlreadyExists:
			utils.SendError(resp, err, http.StatusConflict)
		case connection.ErrConnectionCancelled:
			utils.SendError(resp, err, statusConnectCancelled)
		default:
			log.Error().Err(err).Msg("")
			utils.SendError(resp, err, http.StatusInternalServerError)
		}
		return
	}
	resp.WriteHeader(http.StatusCreated)
	ce.Status(resp, req, params)
}

// Kill stops connection
// swagger:operation DELETE /connection Connection connectionCancel
// ---
// summary: Stops connection
// description: Stops current connection
// responses:
//   202:
//     description: Connection Stopped
//   409:
//     description: Conflict. No connection exists
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ce *ConnectionEndpoint) Kill(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	err := ce.manager.Disconnect()
	if err != nil {
		switch err {
		case connection.ErrNoConnection:
			utils.SendError(resp, err, http.StatusConflict)
		default:
			utils.SendError(resp, err, http.StatusInternalServerError)
		}
		return
	}
	resp.WriteHeader(http.StatusAccepted)
}

// GetStatistics returns statistics about current connection
// swagger:operation GET /connection/statistics Connection connectionStatistics
// ---
// summary: Returns connection statistics
// description: Returns statistics about current connection
// responses:
//   200:
//     description: Connection statistics
//     schema:
//       "$ref": "#/definitions/ConnectionStatisticsDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ce *ConnectionEndpoint) GetStatistics(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	connection := ce.stateProvider.GetState().Connection
	response := contract.NewConnectionStatisticsDTO(connection.Session, connection.Statistics, connection.Throughput, connection.Invoice)

	utils.WriteAsJSON(response, writer)
}

// AddRoutesForConnection adds connections routes to given router
func AddRoutesForConnection(router *httprouter.Router, manager connection.Manager,
	stateProvider stateProvider, proposalRepository proposal.Repository, identityRegistry identityRegistry) {
	connectionEndpoint := NewConnectionEndpoint(manager, stateProvider, proposalRepository, identityRegistry)
	router.GET("/connection", connectionEndpoint.Status)
	router.PUT("/connection", connectionEndpoint.Create)
	router.DELETE("/connection", connectionEndpoint.Kill)
	router.GET("/connection/statistics", connectionEndpoint.GetStatistics)
}

func toConnectionRequest(req *http.Request) (*connectionRequest, error) {
	var connectionRequest = connectionRequest{
		// This defaults the service type to openvpn, for backward compatibility
		// If specified in the request, the value will get overridden
		ServiceType: "openvpn",
	}
	err := json.NewDecoder(req.Body).Decode(&connectionRequest)
	if err != nil {
		return nil, err
	}
	return &connectionRequest, nil
}

func getConnectOptions(cr *connectionRequest) connection.ConnectParams {
	dns := connection.DNSOptionAuto
	if cr.ConnectOptions.DNS != "" {
		dns = cr.ConnectOptions.DNS
	}
	return connection.ConnectParams{
		DisableKillSwitch: cr.ConnectOptions.DisableKillSwitch,
		DNS:               dns,
	}
}

func validateConnectionRequest(cr *connectionRequest) *validation.FieldErrorMap {
	errs := validation.NewErrorMap()
	if len(cr.ConsumerID) == 0 {
		errs.ForField("consumer_id").AddError("required", "Field is required")
	}
	if len(cr.ProviderID) == 0 {
		errs.ForField("provider_id").AddError("required", "Field is required")
	}
	if len(cr.AccountantID) == 0 {
		errs.ForField("accountant_id").AddError("required", "Field is required")
	}
	return errs
}

var emptyAddress = common.Address{}

func toConnectionResponse(status connection.Status) connectionResponse {
	response := connectionResponse{
		Status:     string(status.State),
		SessionID:  string(status.SessionID),
		ConsumerID: status.ConsumerID.Address,
	}
	if status.AccountantID != emptyAddress {
		response.AccountantID = status.AccountantID.Hex()
	}

	if status.Proposal.ProviderID != "" {
		proposalRes := proposalToRes(status.Proposal)
		response.Proposal = proposalRes
	}
	return response
}
