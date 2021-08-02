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
	"fmt"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/market"
)

// NewProposalDTO maps to API service proposal.
func NewProposalDTO(p proposal.PricedServiceProposal) ProposalDTO {
	return ProposalDTO{
		Format:         p.Format,
		Compatibility:  p.Compatibility,
		ProviderID:     p.ProviderID,
		ServiceType:    p.ServiceType,
		Location:       NewServiceLocationsDTO(p.Location),
		AccessPolicies: p.AccessPolicies,
		Quality: Quality{
			Quality:   p.Quality.Quality,
			Latency:   p.Quality.Latency,
			Bandwidth: p.Quality.Bandwidth,
		},
		Price: Price{
			Currency: "MYSTT",
			PerHour:  p.Price.PricePerHour.Uint64(),
			PerGiB:   p.Price.PricePerGiB.Uint64(),
		},
	}
}

// NewServiceLocationsDTO maps to API service location.
func NewServiceLocationsDTO(l market.Location) ServiceLocationDTO {
	return ServiceLocationDTO{
		Continent: l.Continent,
		Country:   l.Country,
		City:      l.City,
		ASN:       l.ASN,
		ISP:       l.ISP,
		IPType:    l.IPType,
	}
}

// ListProposalsResponse holds list of proposals.
// swagger:model ListProposalsResponse
type ListProposalsResponse struct {
	Proposals []ProposalDTO `json:"proposals"`
}

// ProposalDTO holds service proposal details.
// swagger:model ProposalDTO
type ProposalDTO struct {
	// Proposal format.
	Format string `json:"format"`

	// Compatibility level.
	Compatibility int `json:"compatibility"`

	// provider who offers service
	// example: 0x0000000000000000000000000000000000000001
	ProviderID string `json:"provider_id"`

	// type of service provider offers
	// example: openvpn
	ServiceType string `json:"service_type"`

	// Service location
	Location ServiceLocationDTO `json:"location"`

	// Service price
	Price Price `json:"price"`

	// AccessPolicies
	AccessPolicies *[]market.AccessPolicy `json:"access_policies,omitempty"`

	// Quality of the service.
	Quality Quality `json:"quality"`
}

// Price represents the service price.
// swagger:model Price
type Price struct {
	Currency string `json:"currency"`
	PerHour  uint64 `json:"per_hour"`
	PerGiB   uint64 `json:"per_gib"`
}

func (p ProposalDTO) String() string {
	return fmt.Sprintf("Provider: %s, ServiceType: %s, Country: %s", p.ProviderID, p.ServiceType, p.Location.Country)
}

// ListProposalFilterPresetsResponse holds a list of proposal filter presets.
// swagger:model ListProposalFilterPresetsResponse
type ListProposalFilterPresetsResponse struct {
	Items []FilterPreset `json:"items"`
}

// FilterPreset is a pre-defined proposal filter.
// swagger:model FilterPreset
type FilterPreset struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// NewFilterPreset maps to the FilterPreset.
func NewFilterPreset(preset proposal.FilterPreset) FilterPreset {
	return FilterPreset{
		ID:   preset.ID,
		Name: preset.Name,
	}
}

// ServiceLocationDTO holds service location metadata.
// swagger:model ServiceLocationDTO
type ServiceLocationDTO struct {
	// example: EU
	Continent string `json:"continent,omitempty"`
	// example: NL
	Country string `json:"country,omitempty"`
	// example: Amsterdam
	City string `json:"city,omitempty"`

	// Autonomous System Number
	// example: 00001
	ASN int `json:"asn"`
	// example: Telia Lietuva, AB
	ISP string `json:"isp,omitempty"`
	// example: residential
	IPType string `json:"ip_type,omitempty"`
}

// Quality holds proposal quality metrics.
// swagger:model Quality
type Quality struct {
	Quality   float64 `json:"quality"`
	Latency   float64 `json:"latency"`
	Bandwidth float64 `json:"bandwidth"`
}
