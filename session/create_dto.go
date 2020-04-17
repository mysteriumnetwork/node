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

package session

import (
	"encoding/json"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/identity"
)

const endpointSessionCreate = communication.RequestEndpoint("session-create")

var (
	responseInvalidProposal    = CreateResponse{Success: false, Message: "Invalid Proposal"}
	responseUnsupportedVersion = CreateResponse{Success: false, Message: "You are running and old version, please update your software"}
	responseInternalError      = CreateResponse{Success: false, Message: "Internal Error"}
)

// CreateRequest structure represents message from service consumer to initiate session for given proposal id
type CreateRequest struct {
	ProposalID   int             `json:"proposal_id"`
	Config       json.RawMessage `json:"config"`
	ConsumerInfo *ConsumerInfo   `json:"consumer_info,omitempty"`
}

// ConsumerInfo represents the consumer related information
type ConsumerInfo struct {
	// TODO Should not use internal structures for transport
	IssuerID       identity.Identity `json:"issuerID"`
	AccountantID   identity.Identity `json:"accountantID"`
	PaymentVersion PaymentVersion    `json:"paymentVersion"`
}

// CreateResponse structure represents service provider response to given session request from consumer
type CreateResponse struct {
	Success     bool        `json:"success"`
	Message     string      `json:"message"`
	Session     SessionDto  `json:"session"`
	PaymentInfo PaymentInfo `json:"paymentInfo"`
}

// SessionDto structure represents session information data within session creation response (session id and configuration options for underlying service type)
type SessionDto struct {
	ID     ID              `json:"id"`
	Config json.RawMessage `json:"config"`
}

// PaymentInfo represents the payment version information
type PaymentInfo struct {
	Supports string `json:"supported"`
}

// PaymentVersion represents the different payment versions we have
type PaymentVersion string

// PaymentVersionV3 represents the new pingpong version
const PaymentVersionV3 PaymentVersion = "v3"
