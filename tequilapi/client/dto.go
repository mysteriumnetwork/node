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

package client

import (
	"encoding/json"
	"fmt"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/money"
)

// Fees represents the transactor fee
type Fees struct {
	Registration uint64 `json:"registration"`
	Settlement   uint64 `json:"settlement"`
}

// StatusDTO holds connection status and session id
type StatusDTO struct {
	ConsumerID string      `json:"consumer_id"`
	Status     string      `json:"status"`
	SessionID  string      `json:"session_id"`
	Proposal   ProposalDTO `json:"proposal"`
}

// StatisticsDTO holds statistics about connection
type StatisticsDTO struct {
	BytesSent     uint64 `json:"bytes_sent"`
	BytesReceived uint64 `json:"bytes_received"`
	Duration      int    `json:"duration"`
	TokensSpent   uint64 `json:"tokens_spent"`
}

// ProposalList describes list of proposals
type ProposalList struct {
	Proposals []ProposalDTO `json:"proposals"`
}

// ProposalDTO describes service proposal
type ProposalDTO struct {
	ID                int                  `json:"id"`
	ProviderID        string               `json:"provider_id"`
	ServiceType       string               `json:"service_type"`
	ServiceDefinition ServiceDefinitionDTO `json:"service_definition"`
	AccessPolicies    []AccessPolicy       `json:"access_policies"`
	PaymentMethodType string               `json:"payment_method_type"`
	PaymentMethod     paymentMethodRes     `json:"payment_method"`
}

type paymentMethodRes struct {
	Type  string         `json:"type"`
	Price money.Money    `json:"price"`
	Rate  paymentRateRes `json:"rate"`
}

type paymentRateRes struct {
	PerSeconds uint64 `json:"per_seconds"`
	PerBytes   uint64 `json:"per_bytes"`
}

// AccessPolicy represents the access controls for proposal
type AccessPolicy struct {
	ID     string `json:"id"`
	Source string `json:"source"`
}

func (p ProposalDTO) String() string {
	return fmt.Sprintf("Id: %d , Provider: %s, Country: %s", p.ID, p.ProviderID, p.ServiceDefinition.LocationOriginate.Country)
}

// ServiceDefinitionDTO describes service of proposal
type ServiceDefinitionDTO struct {
	LocationOriginate ServiceLocationDTO `json:"location_originate"`
}

// ServiceLocationDTO describes location of proposal
type ServiceLocationDTO struct {
	Country string `json:"country"`
}

// HealthcheckDTO holds returned healthcheck response
type HealthcheckDTO struct {
	Uptime    string       `json:"uptime"`
	Process   int          `json:"process"`
	Version   string       `json:"version"`
	BuildInfo BuildInfoDTO `json:"build_info"`
}

// BuildInfoDTO holds info about build
type BuildInfoDTO struct {
	Commit      string `json:"commit"`
	Branch      string `json:"branch"`
	BuildNumber string `json:"build_number"`
}

// LocationDTO describes location metadata
type LocationDTO struct {
	IP  string `json:"ip"`
	ASN int    `json:"asn"`
	ISP string `json:"isp"`

	Continent string `json:"continent"`
	Country   string `json:"country"`
	City      string `json:"city"`

	UserType string `json:"user_type"`
}

// RegistrationDataDTO holds input data required to register new myst identity on blockchain smart contract
type RegistrationDataDTO struct {
	Status     string `json:"status"`
	Registered bool   `json:"registered"`
}

// ConnectOptions copied from tequilapi endpoint
type ConnectOptions struct {
	DisableKillSwitch bool                 `json:"kill_switch"`
	DNS               connection.DNSOption `json:"dns"`
}

// ConnectionSessionListDTO copied from tequilapi endpoint
type ConnectionSessionListDTO struct {
	Sessions []ConnectionSessionDTO `json:"sessions"`
}

// ConnectionSessionDTO copied from tequilapi endpoint
type ConnectionSessionDTO struct {
	SessionID       string `json:"session_id"`
	ProviderID      string `json:"provider_id"`
	ServiceType     string `json:"service_type"`
	ProviderCountry string `json:"provider_country"`
	DateStarted     string `json:"date_started"`
	BytesSent       uint64 `json:"bytes_sent"`
	BytesReceived   uint64 `json:"bytes_received"`
	Duration        uint64 `json:"duration"`
	Status          string `json:"status"`
}

// ServiceListDTO represents a list of running services on the node
type ServiceListDTO []ServiceInfoDTO

// ServiceInfoDTO represents running service information
type ServiceInfoDTO struct {
	ID          string          `json:"id"`
	ProviderID  string          `json:"provider_id"`
	ServiceType string          `json:"type"`
	Options     json.RawMessage `json:"options"`
	Status      string          `json:"status"`
	Proposal    ProposalDTO     `json:"proposal"`
}

// ServiceSessionListDTO copied from tequilapi endpoint
type ServiceSessionListDTO struct {
	Sessions []ServiceSessionDTO `json:"sessions"`
}

// ServiceSessionDTO copied from tequilapi endpoint
type ServiceSessionDTO struct {
	ID         string `json:"id"`
	ConsumerID string `json:"consumer_id"`
}

// AccessPoliciesRequest represents the access controls for service start
type AccessPoliciesRequest struct {
	IDs []string `json:"ids"`
}

// NATStatusDTO gives information about NAT traversal success or failure
type NATStatusDTO struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// SettleRequest represents the request to settle accountant promises
type SettleRequest struct {
	AccountantID string `json:"accountant_id"`
	ProviderID   string `json:"provider_id"`
}
