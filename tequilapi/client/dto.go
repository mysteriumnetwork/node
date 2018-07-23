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

import "fmt"

// StatusDTO holds connection status and session id
type StatusDTO struct {
	Status    string `json:"status"`
	SessionID string `json:"sessionId"`
}

// StatisticsDTO holds statistics about connection
type StatisticsDTO struct {
	BytesSent     int `json:"bytesSent"`
	BytesReceived int `json:"bytesReceived"`
	Duration      int `json:"duration"`
}

// ProposalList describes list of proposals
type ProposalList struct {
	Proposals []ProposalDTO `json:"proposals"`
}

// ProposalDTO describes service proposal
type ProposalDTO struct {
	ID                int                  `json:"id"`
	ProviderID        string               `json:"providerId"`
	ServiceDefinition ServiceDefinitionDTO `json:"serviceDefinition"`
}

func (p ProposalDTO) String() string {
	return fmt.Sprintf("Id: %d , Provider: %s, Country: %s", p.ID, p.ProviderID, p.ServiceDefinition.LocationOriginate.Country)
}

// ServiceDefinitionDTO describes service of proposal
type ServiceDefinitionDTO struct {
	LocationOriginate LocationDTO `json:"locationOriginate"`
}

// LocationDTO describes location
type LocationDTO struct {
	Country string `json:"country"`
}

// IdentityDTO holds identity address
type IdentityDTO struct {
	Address string `json:"id"`
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

// RegistrationDataDTO holds input data required to register new myst identity on blockchain smart contract
type RegistrationStatusDTO struct {
	Registered bool
	PublicKey  PublicKeyPartsDTO
	Signature  SignatureDTO
}

// PublicKeyPartsDTO holds public key parts in hex, split into 32 byte blocks
type PublicKeyPartsDTO struct {
	Part1 string
	Part2 string
}

// SignatureDTO holds decomposed ECDSA signature values R, S and V
type SignatureDTO struct {
	R string
	S string
	V uint8
}
