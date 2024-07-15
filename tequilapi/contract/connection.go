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
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/mysteriumnetwork/node/consumer/bandwidth"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/payments/crypto"
)

var emptyAddress = common.Address{}

// NewConnectionInfoDTO maps to API connection status.
func NewConnectionInfoDTO(session connectionstate.Status) ConnectionInfoDTO {
	response := ConnectionInfoDTO{
		Status:     string(session.State),
		ConsumerID: session.ConsumerID.Address,
		SessionID:  string(session.SessionID),
	}
	if session.HermesID != emptyAddress {
		response.HermesID = session.HermesID.Hex()
	}
	// None exists, for not started connection
	if session.Proposal.ProviderID != "" {
		proposalRes := NewProposalDTO(session.Proposal)
		response.Proposal = &proposalRes
	}
	return response
}

// ConnectionDiagInfoDTO holds provider check result
// swagger:model ConnectionDiagInfoDTO
type ConnectionDiagInfoDTO struct {
	ProviderID string `json:"provider_id"`
	Error      string `json:"error"`
	DiagError  string `json:"diag_err"`
}

// ConnectionInfoDTO holds partial consumer connection details.
// swagger:model ConnectionInfoDTO
type ConnectionInfoDTO struct {
	// example: Connected
	Status string `json:"status"`

	// example: 0x00
	ConsumerID string `json:"consumer_id,omitempty"`

	// example: 0x00
	HermesID string `json:"hermes_id,omitempty"`

	// example: {"id":1,"provider_id":"0x71ccbdee7f6afe85a5bc7106323518518cd23b94","service_type":"openvpn","location":{"country":"CA"}}
	Proposal *ProposalDTO `json:"proposal,omitempty"`

	// example: 4cfb0324-daf6-4ad8-448b-e61fe0a1f918
	SessionID string `json:"session_id,omitempty"`
}

// NewConnectionDTO maps to API connection.
func NewConnectionDTO(session connectionstate.Status, statistics connectionstate.Statistics, throughput bandwidth.Throughput, invoice crypto.Invoice) ConnectionDTO {
	dto := ConnectionDTO{
		ConnectionInfoDTO: NewConnectionInfoDTO(session),
	}
	if !statistics.At.IsZero() {
		statsDto := NewConnectionStatisticsDTO(session, statistics, throughput, invoice)
		dto.Statistics = &statsDto
	}
	return dto
}

// ConnectionDTO holds full consumer connection details.
// swagger:model ConnectionDTO
type ConnectionDTO struct {
	ConnectionInfoDTO
	Statistics *ConnectionStatisticsDTO `json:"statistics,omitempty"`
}

// NewConnectionStatisticsDTO maps to API connection stats.
func NewConnectionStatisticsDTO(session connectionstate.Status, statistics connectionstate.Statistics, throughput bandwidth.Throughput, invoice crypto.Invoice) ConnectionStatisticsDTO {
	agreementTotal := new(big.Int)
	if invoice.AgreementTotal != nil {
		agreementTotal = invoice.AgreementTotal
	}
	return ConnectionStatisticsDTO{
		Duration:           int(session.Duration().Seconds()),
		BytesSent:          statistics.BytesSent,
		BytesReceived:      statistics.BytesReceived,
		ThroughputSent:     datasize.BitSize(throughput.Up).Bits(),
		ThroughputReceived: datasize.BitSize(throughput.Down).Bits(),
		TokensSpent:        agreementTotal,
		SpentTokens:        NewTokens(agreementTotal),
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
	TokensSpent *big.Int `json:"tokens_spent"`

	SpentTokens Tokens `json:"spent_tokens"`
}

// ConnectionTrafficDTO holds consumer connection traffic information.
// swagger:model ConnectionTrafficDTO
type ConnectionTrafficDTO struct {
	// example: 1024
	BytesSent uint64 `json:"bytes_sent"`

	// example: 1024
	BytesReceived uint64 `json:"bytes_received"`
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

	Filter ConnectionCreateFilter `json:"filter"`

	// hermes identity
	// example: 0x0000000000000000000000000000000000000003
	HermesID string `json:"hermes_id"`

	// service type. Possible values are "openvpn", "wireguard" and "noop"
	// required: false
	// default: openvpn
	// example: openvpn
	ServiceType string `json:"service_type"`

	// connect options
	// required: false
	ConnectOptions ConnectOptions `json:"connect_options,omitempty"`
}

// ConnectionCreateFilter describes filter for the connection request to lookup
// for a requested proposals based on specified params.
type ConnectionCreateFilter struct {
	Providers               []string `json:"providers,omitempty"`
	CountryCode             string   `json:"country_code,omitempty"`
	IPType                  string   `json:"ip_type,omitempty"`
	IncludeMonitoringFailed bool     `json:"include_monitoring_failed,omitempty"`
	SortBy                  string   `json:"sort_by,omitempty"`
}

// Validate validates fields in request.
func (cr ConnectionCreateRequest) Validate() *apierror.APIError {
	v := apierror.NewValidator()
	if len(cr.ConsumerID) == 0 {
		v.Required("consumer_id")
	}
	return v.Err()
}

// Event creates a quality connection event to be send as a quality metric.
func (cr ConnectionCreateRequest) Event(stage string, errMsg string) quality.ConnectionEvent {
	return quality.ConnectionEvent{
		ServiceType: cr.ServiceType,
		ProviderID:  cr.ProviderID,
		ConsumerID:  cr.ConsumerID,
		HermesID:    cr.HermesID,
		Error:       errMsg,
		Stage:       stage,
	}
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

	ProxyPort int `json:"proxy_port"`
}
