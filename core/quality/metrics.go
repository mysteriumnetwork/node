/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package quality

const (
	// StagePraseRequest describes connection request parse event.
	StagePraseRequest = "parse_request"
	// StageValidateRequest describes connection request validate event.
	StageValidateRequest = "validate_request"

	// StageNoProposal describes connection request event when proposal does not exists.
	StageNoProposal = "no_proposal"
	// StageGetProposal describes proposal get event for connection create.
	StageGetProposal = "get_proposal"

	// StageConnectionOK describes successful connection event.
	StageConnectionOK = "connection_ok"
	// StageConnectionCanceled describes canceled connection event.
	StageConnectionCanceled = "connection_canceled"
	// StageConnectionAlreadyExists describes already exists connection event.
	StageConnectionAlreadyExists = "connection_already_exists"
	// StageConnectionUnknownError describes unknown connection event.
	StageConnectionUnknownError = "connection_unknown_error"

	// StageRegistrationGetStatus describes getting registration status event.
	StageRegistrationGetStatus = "registration_get_status"
	// StageRegistrationUnregistered describes registration unregistered status event.
	StageRegistrationUnregistered = "registration_unregistered"
	// StageRegistrationInProgress describes registration in progress status event.
	StageRegistrationInProgress = "registration_in_progress"
	// StageRegistrationRegistered describes registration registered status event.
	StageRegistrationRegistered = "registration_registered"
	// StageRegistrationUnknown describes registration unknown status event.
	StageRegistrationUnknown = "registration_unknown"
)

// ServiceMetricsResponse represents response from the quality oracle service.
type ServiceMetricsResponse struct {
	Connects []ConnectMetric `json:"connects"`
}

// ConnectMetric represents a proposal with quality info.
type ConnectMetric struct {
	ProposalID       ProposalID   `json:"proposalId"`
	ConnectCount     ConnectCount `json:"connectCount"`
	MonitoringFailed bool         `json:"monitoringFailed"`
}

// ProposalID represents the struct used to uniquely identify proposals.
type ProposalID struct {
	ProviderID  string `json:"providerId" example:"0x286f0e9eb943eca95646bf4933698856579b096e"`
	ServiceType string `json:"serviceType" example:"openvpn"`
}

// ConnectCount represents the connection count statistics.
type ConnectCount struct {
	Success int `json:"success" example:"100" format:"int64"`
	Fail    int `json:"fail" example:"50" format:"int64"`
	Timeout int `json:"timeout" example:"10" format:"int64"`
}

// ConnectionEvent represents the connection stages events.
type ConnectionEvent struct {
	ServiceType string `json:"service_type"`
	ProviderID  string `json:"provider_id"`
	ConsumerID  string `json:"consumer_id"`
	HermesID    string `json:"hermes_id"`
	Error       string `json:"error"`
	Stage       string `json:"stage"`
}

// AppTopicConnectionEvents represents event bus topic for the connection events.
const AppTopicConnectionEvents = "connection_events"
