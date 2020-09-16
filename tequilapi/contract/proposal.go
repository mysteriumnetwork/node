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

	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
)

// NewProposalDTO maps to API service proposal.
func NewProposalDTO(p market.ServiceProposal) ProposalDTO {
	return ProposalDTO{
		ID:                p.ID,
		ProviderID:        p.ProviderID,
		ServiceType:       p.ServiceType,
		ServiceDefinition: NewServiceDefinitionDTO(p.ServiceDefinition),
		AccessPolicies:    p.AccessPolicies,
		PaymentMethod:     NewPaymentMethodDTO(p.PaymentMethod),
	}
}

// NewPaymentMethodDTO maps to API payment method.
func NewPaymentMethodDTO(m market.PaymentMethod) PaymentMethodDTO {
	if m == nil {
		return PaymentMethodDTO{}
	}
	return PaymentMethodDTO{
		Type:  m.GetType(),
		Price: m.GetPrice(),
		Rate: PaymentRateDTO{
			PerSeconds: uint64(m.GetRate().PerTime.Seconds()),
			PerBytes:   m.GetRate().PerByte,
		},
	}
}

// NewServiceDefinitionDTO maps to API service definition.
func NewServiceDefinitionDTO(s market.ServiceDefinition) ServiceDefinitionDTO {
	if s == nil {
		return ServiceDefinitionDTO{}
	}
	return ServiceDefinitionDTO{
		LocationOriginate: NewLocationsDTO(s.GetLocation()),
	}
}

// NewLocationsDTO maps to API service location.
func NewLocationsDTO(l market.Location) ServiceLocationDTO {
	return ServiceLocationDTO{
		Continent: l.Continent,
		Country:   l.Country,
		City:      l.City,

		ASN:      l.ASN,
		ISP:      l.ISP,
		NodeType: l.NodeType,
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
	// per provider unique serial number of service description provided
	// example: 5
	ID int `json:"id"`

	// provider who offers service
	// example: 0x0000000000000000000000000000000000000001
	ProviderID string `json:"provider_id"`

	// type of service provider offers
	// example: openvpn
	ServiceType string `json:"service_type"`

	// qualitative service definition
	ServiceDefinition ServiceDefinitionDTO `json:"service_definition"`

	// Metrics of the service
	Metrics *ProposalMetricsDTO `json:"metrics,omitempty"`

	// AccessPolicies
	AccessPolicies *[]market.AccessPolicy `json:"access_policies,omitempty"`

	// PaymentMethod
	PaymentMethod PaymentMethodDTO `json:"payment_method"`
}

func (p ProposalDTO) String() string {
	return fmt.Sprintf("Id: %d , Provider: %s, Country: %s", p.ID, p.ProviderID, p.ServiceDefinition.LocationOriginate.Country)
}

// ServiceDefinitionDTO holds specific service details.
// swagger:model ServiceDefinitionDTO
type ServiceDefinitionDTO struct {
	LocationOriginate ServiceLocationDTO `json:"location_originate"`
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
	NodeType string `json:"node_type,omitempty"`
}

// ProposalMetricsDTO holds proposal quality metrics from Quality Oracle.
// swagger:model ProposalMetricsDTO
type ProposalMetricsDTO struct {
	ConnectCount     ProposalMetricConnectsDTO `json:"connect_count"`
	MonitoringFailed bool                      `json:"monitoring_failed"`
}

// ProposalMetricConnectsDTO represents the metric for connect stats.
// swagger:model ProposalMetricConnectsDTO
type ProposalMetricConnectsDTO struct {
	Success int `json:"success" example:"100" format:"int64"`
	Fail    int `json:"fail" example:"50" format:"int64"`
	Timeout int `json:"timeout" example:"10" format:"int64"`
}

// PaymentMethodDTO holds payment method details.
// swagger:model PaymentMethodDTO
type PaymentMethodDTO struct {
	Type  string         `json:"type"`
	Price money.Money    `json:"price"`
	Rate  PaymentRateDTO `json:"rate"`
}

// PaymentRateDTO holds payment frequencies.
// swagger:model PaymentRateDTO
type PaymentRateDTO struct {
	PerSeconds uint64 `json:"per_seconds"`
	PerBytes   uint64 `json:"per_bytes"`
}

// ProposalsQualityMetricsResponse holds all quality metrics.
// swagger:model QualityMetricsDTO
type ProposalsQualityMetricsResponse struct {
	Metrics []QualityMetricsResponse `json:"metrics"`
}

// QualityMetricsResponse holds quality metrics per service.
// swagger:model QualityMetricsResponse
type QualityMetricsResponse struct {
	ProviderID  string `json:"provider_id"`
	ServiceType string `json:"service_type"`
	ProposalMetricsDTO
}
