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
)

// Fees represents the transactor fee
type Fees struct {
	Registration uint64 `json:"registration"`
	Settlement   uint64 `json:"settlement"`
}

// StatusDTO holds connection status and session id
type StatusDTO struct {
	Status    string      `json:"status"`
	SessionID string      `json:"sessionId"`
	Proposal  ProposalDTO `json:"proposal"`
}

// StatisticsDTO holds statistics about connection
type StatisticsDTO struct {
	BytesSent     uint64 `json:"bytesSent"`
	BytesReceived uint64 `json:"bytesReceived"`
	Duration      int    `json:"duration"`
}

// ProposalList describes list of proposals
type ProposalList struct {
	Proposals []ProposalDTO `json:"proposals"`
}

// ProposalDTO describes service proposal
type ProposalDTO struct {
	ID                int                  `json:"id"`
	ProviderID        string               `json:"providerId"`
	ServiceType       string               `json:"serviceType"`
	ServiceDefinition ServiceDefinitionDTO `json:"serviceDefinition"`
}

func (p ProposalDTO) String() string {
	return fmt.Sprintf("Id: %d , Provider: %s, Country: %s", p.ID, p.ProviderID, p.ServiceDefinition.LocationOriginate.Country)
}

// ServiceDefinitionDTO describes service of proposal
type ServiceDefinitionDTO struct {
	LocationOriginate ServiceLocationDTO `json:"locationOriginate"`
}

// ServiceLocationDTO describes location of proposal
type ServiceLocationDTO struct {
	Country string `json:"country"`
}

// IdentityDTO holds identity address
type IdentityDTO struct {
	Address string `json:"id"`
}

// IdentityStatusDTO holds identity status with balance
type IdentityStatusDTO struct {
	ChannelAddress string `json:"channel_address"`
	IsRegistered   bool   `json:"is_registered"`
	Balance        uint64 `json:"balance"`
}

// IdentityList holds returned list of identities
type IdentityList struct {
	Identities []IdentityDTO `json:"identities"`
}

// HealthcheckDTO holds returned healthcheck response
type HealthcheckDTO struct {
	Uptime    string       `json:"uptime"`
	Process   int          `json:"process"`
	Version   string       `json:"version"`
	BuildInfo BuildInfoDTO `json:"buildInfo"`
}

// BuildInfoDTO holds info about build
type BuildInfoDTO struct {
	Commit      string `json:"commit"`
	Branch      string `json:"branch"`
	BuildNumber string `json:"buildNumber"`
}

// LocationDTO describes location metadata
type LocationDTO struct {
	IP  string `json:"ip"`
	ASN int    `json:"asn"`
	ISP string `json:"isp"`

	Continent string `json:"continent"`
	Country   string `json:"country"`
	City      string `json:"city"`

	UserType string `json:"userType"`
}

// RegistrationDataDTO holds input data required to register new myst identity on blockchain smart contract
type RegistrationDataDTO struct {
	Registered bool              `json:"registered"`
	PublicKey  PublicKeyPartsDTO `json:"publicKey"`
	Signature  SignatureDTO      `json:"signature"`
}

// PublicKeyPartsDTO holds public key parts in hex, split into 32 byte blocks
type PublicKeyPartsDTO struct {
	Part1 string `json:"part1"`
	Part2 string `json:"part2"`
}

// SignatureDTO holds decomposed ECDSA signature values R, S and V
type SignatureDTO struct {
	R string `json:"r"`
	S string `json:"s"`
	V uint8  `json:"v"`
}

// ConnectOptions copied from tequilapi endpoint
type ConnectOptions struct {
	DisableKillSwitch bool                 `json:"killSwitch"`
	DNS               connection.DNSOption `json:"dns"`
}

// ConnectionSessionListDTO copied from tequilapi endpoint
type ConnectionSessionListDTO struct {
	Sessions []ConnectionSessionDTO `json:"sessions"`
}

// ConnectionSessionDTO copied from tequilapi endpoint
type ConnectionSessionDTO struct {
	SessionID       string `json:"sessionId"`
	ProviderID      string `json:"providerId"`
	ServiceType     string `json:"serviceType"`
	ProviderCountry string `json:"providerCountry"`
	DateStarted     string `json:"dateStarted"`
	BytesSent       uint64 `json:"bytesSent"`
	BytesReceived   uint64 `json:"bytesReceived"`
	Duration        uint64 `json:"duration"`
	Status          string `json:"status"`
}

// ServiceListDTO represents a list of running services on the node
type ServiceListDTO []ServiceInfoDTO

// ServiceInfoDTO represents running service information
type ServiceInfoDTO struct {
	ID          string          `json:"id"`
	ProviderID  string          `json:"providerId"`
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
	ConsumerID string `json:"consumerId"`
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
