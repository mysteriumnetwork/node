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
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/mysteriumnetwork/node/tequilapi/validation"
	"github.com/pkg/errors"
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
	DisableKillSwitch bool `json:"killSwitch"`
}

// swagger:model ConnectionRequestDTO
type connectionRequest struct {
	// consumer identity
	// required: true
	// example: 0x0000000000000000000000000000000000000001
	ConsumerID string `json:"consumerId"`

	// provider identity
	// required: true
	// example: 0x0000000000000000000000000000000000000002
	ProviderID string `json:"providerId"`

	// service type. Possible values are "openvpn", "wireguard" and "noop"
	// required: false
	// default: openvpn
	// example: openvpn
	ServiceType string `json:"serviceType"`

	// connect options
	// required: false
	ConnectOptions ConnectOptions `json:"connectOptions,omitempty"`
}

// swagger:model ConnectionStatusDTO
type connectionResponse struct {
	// example: Connected
	Status string `json:"status"`

	// example: 4cfb0324-daf6-4ad8-448b-e61fe0a1f918
	SessionID string `json:"sessionId,omitempty"`

	// example: {"id":1,"providerId":"0x71ccbdee7f6afe85a5bc7106323518518cd23b94","serviceType":"openvpn","serviceDefinition":{"locationOriginate":{"asn":"","country":"CA"}}}
	Proposal *proposalRes `json:"proposal,omitempty"`
}

// swagger:model IPDTO
type ipResponse struct {
	// public IP address
	// example: 127.0.0.1
	IP string `json:"ip"`
}

// swagger:model ConnectionStatisticsDTO
type statisticsResponse struct {
	// example: 1024
	BytesSent uint64 `json:"bytesSent"`

	// example: 1024
	BytesReceived uint64 `json:"bytesReceived"`

	// connection duration in seconds
	// example: 60
	Duration int `json:"duration"`
}

// SessionStatisticsTracker represents the session stat keeper
type SessionStatisticsTracker interface {
	Retrieve() consumer.SessionStatistics
	GetSessionDuration() time.Duration
}

// ProposalGetter defines interface to fetch currently active service proposal by id
type ProposalGetter interface {
	GetProposal(id market.ProposalID) (*market.ServiceProposal, error)
}

// ConnectionEndpoint struct represents /connection resource and it's subresources
type ConnectionEndpoint struct {
	manager           connection.Manager
	statisticsTracker SessionStatisticsTracker
	//TODO connection should use concrete proposal from connection params and avoid going to marketplace
	proposalProvider ProposalGetter
}

// NewConnectionEndpoint creates and returns connection endpoint
func NewConnectionEndpoint(manager connection.Manager, statsKeeper SessionStatisticsTracker, proposalProvider ProposalGetter) *ConnectionEndpoint {
	return &ConnectionEndpoint{
		manager:           manager,
		statisticsTracker: statsKeeper,
		proposalProvider:  proposalProvider,
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
//     description: Parameters in body (consumerId, providerId, serviceType) required for creating new connection
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

	errorMap := validateConnectionRequest(cr)
	if errorMap.HasErrors() {
		utils.SendValidationErrorMessage(resp, errorMap)
		return
	}

	// TODO Pass proposal ID directly in request
	proposal, err := ce.proposalProvider.GetProposal(market.ProposalID{
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
	err = ce.manager.Connect(identity.FromAddress(cr.ConsumerID), *proposal, connectOptions)

	if err != nil {
		switch err {
		case connection.ErrAlreadyExists:
			utils.SendError(resp, err, http.StatusConflict)
		case connection.ErrConnectionCancelled:
			utils.SendError(resp, err, statusConnectCancelled)
		default:
			log.Error(err)
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
	st := ce.statisticsTracker.Retrieve()

	duration := ce.statisticsTracker.GetSessionDuration()

	response := statisticsResponse{
		BytesSent:     st.BytesSent,
		BytesReceived: st.BytesReceived,
		Duration:      int(duration.Seconds()),
	}

	utils.WriteAsJSON(response, writer)
}

// AddRoutesForConnection adds connections routes to given router
func AddRoutesForConnection(router *httprouter.Router, manager connection.Manager,
	statsKeeper SessionStatisticsTracker, proposalProvider ProposalGetter) {
	connectionEndpoint := NewConnectionEndpoint(manager, statsKeeper, proposalProvider)
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
	return connection.ConnectParams{DisableKillSwitch: cr.ConnectOptions.DisableKillSwitch}
}

func validateConnectionRequest(cr *connectionRequest) *validation.FieldErrorMap {
	errs := validation.NewErrorMap()
	if len(cr.ConsumerID) == 0 {
		errs.ForField("consumerId").AddError("required", "Field is required")
	}
	if len(cr.ProviderID) == 0 {
		errs.ForField("providerId").AddError("required", "Field is required")
	}
	return errs
}

func toConnectionResponse(status connection.Status) connectionResponse {
	response := connectionResponse{
		Status:    string(status.State),
		SessionID: string(status.SessionID),
	}

	if status.Proposal.ProviderID != "" {
		proposalRes := proposalToRes(status.Proposal)
		response.Proposal = &proposalRes
	}
	return response
}
