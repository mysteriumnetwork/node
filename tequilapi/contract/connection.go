/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package contract

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/consumer/bandwidth"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/node/tequilapi/validation"
	"github.com/mysteriumnetwork/payments/crypto"
)

var emptyAddress = common.Address{}

// NewConnectionStatusDTO maps to API connection status.
func NewConnectionStatusDTO(connection connection.Status) ConnectionStatusDTO {
	response := ConnectionStatusDTO{
		Status:     string(connection.State),
		ConsumerID: connection.ConsumerID.Address,
		SessionID:  string(connection.SessionID),
	}
	if connection.AccountantID != emptyAddress {
		response.AccountantAddress = connection.AccountantID.Hex()
	}
	// nNne exists, for not started connection
	if connection.Proposal.ProviderID != "" {
		proposalRes := NewProposalDTO(connection.Proposal)
		response.Proposal = &proposalRes
	}
	return response
}

// ConnectionStatusDTO holds partial consumer connection details.
// swagger:model ConnectionStatusDTO
type ConnectionStatusDTO struct {
	// example: Connected
	Status string `json:"status"`

	// example: 0x00
	ConsumerID string `json:"consumer_id,omitempty"`

	// example: 0x00
	AccountantAddress string `json:"accountant_address,omitempty"`

	// example: {"id":1,"provider_id":"0x71ccbdee7f6afe85a5bc7106323518518cd23b94","servcie_type":"openvpn","service_definition":{"location_originate":{"asn":"","country":"CA"}}}
	Proposal *ProposalDTO `json:"proposal,omitempty"`

	// example: 4cfb0324-daf6-4ad8-448b-e61fe0a1f918
	SessionID string `json:"session_id,omitempty"`
}

// NewConnectionDTO maps to API connection.
func NewConnectionDTO(connection connection.Status, statistics connection.Statistics, throughput bandwidth.Throughput, invoice crypto.Invoice) ConnectionDTO {
	dto := ConnectionDTO{
		ConnectionStatusDTO: NewConnectionStatusDTO(connection),
	}
	if !statistics.At.IsZero() {
		statsDto := NewConnectionStatisticsDTO(connection, statistics, throughput, invoice)
		dto.Statistics = &statsDto
	}
	return dto
}

// ConnectionDTO holds full consumer connection details.
// swagger:model ConnectionDTO
type ConnectionDTO struct {
	ConnectionStatusDTO
	Statistics *ConnectionStatisticsDTO `json:"statistics,omitempty"`
}

// NewConnectionStatisticsDTO maps to API connection stats.
func NewConnectionStatisticsDTO(connection connection.Status, statistics connection.Statistics, throughput bandwidth.Throughput, invoice crypto.Invoice) ConnectionStatisticsDTO {
	return ConnectionStatisticsDTO{
		Duration:           int(connection.Duration().Seconds()),
		BytesSent:          statistics.BytesSent,
		BytesReceived:      statistics.BytesReceived,
		ThroughputSent:     datasize.BitSize(throughput.Up).Bits(),
		ThroughputReceived: datasize.BitSize(throughput.Down).Bits(),
		TokensSpent:        invoice.AgreementTotal,
	}
}

// ConnectionStatisticsDTO holds consumer connection statistics.
// swagger:model ConnectionStatisticsDTO
type ConnectionStatisticsDTO struct {
	// example: 1024
	BytesSent uint64 `json:"bytes_sent"`

	// example: 1024
	BytesReceived uint64 `json:"bytes_received"`

	// Upload speed in bits per second
	// example: 1024
	ThroughputSent uint64 `json:"throughput_sent"`

	// Download speed in bits per second
	// example: 1024
	ThroughputReceived uint64 `json:"throughput_received"`

	// connection duration in seconds
	// example: 60
	Duration int `json:"duration"`

	// example: 500000
	TokensSpent uint64 `json:"tokens_spent"`
}

// ConnectionCreateRequest request used to start a connection.
// swagger:model ConnectionCreateRequestDTO
type ConnectionCreateRequest struct {
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

// Validate validates fields in request
func (cr ConnectionCreateRequest) Validate() *validation.FieldErrorMap {
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
