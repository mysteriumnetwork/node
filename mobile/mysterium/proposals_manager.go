/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package mysterium

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/services/wireguard"
)

const (
	qualityLevelMedium = 1
	qualityLevelHigh   = 2
)

// AutoNATType passed as NATCompatibility parameter in proposal request
// indicates NAT type should be probed automatically immediately within given
// request
const AutoNATType = "auto"

type proposalQualityLevel int

const (
	proposalQualityLevelUnknown proposalQualityLevel = 0
	proposalQualityLevelLow     proposalQualityLevel = 1
	proposalQualityLevelMedium  proposalQualityLevel = 2
	proposalQualityLevelHigh    proposalQualityLevel = 3
)

// GetProposalsRequest represents proposals request.
type GetProposalsRequest struct {
	ServiceType      string
	LocationCountry  string
	IPType           string
	Refresh          bool
	PriceHourMax     float64
	PriceGiBMax      float64
	QualityMin       float32
	PresetID         int
	NATCompatibility string
}

func (r GetProposalsRequest) toFilter() *proposal.Filter {
	return &proposal.Filter{
		ServiceType:        r.ServiceType,
		LocationCountry:    r.LocationCountry,
		IPType:             r.IPType,
		QualityMin:         r.QualityMin,
		ExcludeUnsupported: true,
		NATCompatibility:   nat.NATType(r.NATCompatibility),
	}
}

// GetProposalRequest represents proposal request.
type GetProposalRequest struct {
	ProviderID  string
	ServiceType string
}

type proposalDTO struct {
	ProviderID   string               `json:"provider_id"`
	ServiceType  string               `json:"service_type"`
	Country      string               `json:"country"`
	IPType       string               `json:"ip_type"`
	QualityLevel proposalQualityLevel `json:"quality_level"`
	Price        proposalPrice        `json:"price"`
}

type proposalPrice struct {
	Currency string  `json:"currency"`
	PerGiB   float64 `json:"per_gib"`
	PerHour  float64 `json:"per_hour"`
}

type getProposalsResponse struct {
	Proposals []*proposalDTO `json:"proposals"`
}

type getCountriesResponse map[string]int

type getProposalResponse struct {
	Proposal *proposalDTO `json:"proposal"`
}

type qualityFinder interface {
	ProposalsQuality() []quality.ProposalQuality
}

type proposalRepository interface {
	Proposals(filter *proposal.Filter) ([]proposal.PricedServiceProposal, error)
	Countries(filter *proposal.Filter) (map[string]int, error)
	Proposal(market.ProposalID) (*proposal.PricedServiceProposal, error)
}

type natProber interface {
	Probe(context.Context) (nat.NATType, error)
}

func newProposalsManager(
	repository proposalRepository,
	filterPresetStorage *proposal.FilterPresetStorage,
	natProber natProber,
	cacheTTL time.Duration,
) *proposalsManager {
	return &proposalsManager{
		repository:          repository,
		filterPresetStorage: filterPresetStorage,
		cacheTTL:            cacheTTL,
		natProber:           natProber,
	}
}

type proposalsManager struct {
	repository          proposalRepository
	cache               []proposal.PricedServiceProposal
	cachedAt            time.Time
	cacheTTL            time.Duration
	filterPresetStorage *proposal.FilterPresetStorage
	natProber           natProber
}

func (m *proposalsManager) isCacheStale() bool {
	return time.Now().After(m.cachedAt.Add(m.cacheTTL))
}

func (m *proposalsManager) getCountries(req *GetProposalsRequest) (getCountriesResponse, error) {
	return m.getCountriesFromRepository(req)
}

func (m *proposalsManager) getProposals(req *GetProposalsRequest) (*getProposalsResponse, error) {
	// Get proposals from cache if exists.
	if req.Refresh || m.isCacheStale() {
		apiProposals, err := m.getFromRepository(req)
		if err != nil {
			return nil, err
		}
		m.addToCache(apiProposals)
	}

	filteredProposals, err := m.applyFilter(req.PresetID, m.getFromCache())
	if err != nil {
		return nil, err
	}
	return m.map2Response(filteredProposals)
}

func (m *proposalsManager) applyFilter(presetID int, proposals []proposal.PricedServiceProposal) ([]proposal.PricedServiceProposal, error) {
	if presetID != 0 {
		preset, err := m.filterPresetStorage.Get(presetID)
		if err != nil {
			return nil, err
		}
		return preset.Filter(proposals), nil
	}

	return proposals, nil
}

func (m *proposalsManager) getFromCache() []proposal.PricedServiceProposal {
	return m.cache
}

func (m *proposalsManager) addToCache(proposals []proposal.PricedServiceProposal) {
	m.cache = proposals
	m.cachedAt = time.Now()
}

func (m *proposalsManager) getFromRepository(req *GetProposalsRequest) ([]proposal.PricedServiceProposal, error) {
	filter := req.toFilter()
	if filter.NATCompatibility == AutoNATType {
		natType, err := m.natProber.Probe(context.TODO())
		if err != nil {
			filter.NATCompatibility = ""
		} else {
			filter.NATCompatibility = natType
		}
	}
	allProposals, err := m.repository.Proposals(filter)
	if err != nil {
		return nil, fmt.Errorf("could not get proposals from repository: %w", err)
	}

	// Ideally api should allow to pass multiple service types to skip noop
	// proposals, but for now just filter in memory.
	var res []proposal.PricedServiceProposal
	for _, p := range allProposals {
		if p.ServiceType == openvpn.ServiceType || p.ServiceType == wireguard.ServiceType {
			res = append(res, p)
		}
	}
	return res, nil
}

func (m *proposalsManager) getCountriesFromRepository(req *GetProposalsRequest) (getCountriesResponse, error) {
	filter := req.toFilter()
	if filter.NATCompatibility == AutoNATType {
		natType, err := m.natProber.Probe(context.TODO())
		if err != nil {
			filter.NATCompatibility = ""
		} else {
			filter.NATCompatibility = natType
		}
	}
	countries, err := m.repository.Countries(filter)
	if err != nil {
		return nil, fmt.Errorf("could not get proposals from repository: %w", err)
	}

	return countries, nil
}

func (m *proposalsManager) map2Response(serviceProposals []proposal.PricedServiceProposal) (*getProposalsResponse, error) {
	var proposals []*proposalDTO
	for _, p := range serviceProposals {
		proposals = append(proposals, m.mapProposal(&p))
	}
	return &getProposalsResponse{Proposals: proposals}, nil
}

func (m *proposalsManager) mapProposal(p *proposal.PricedServiceProposal) *proposalDTO {
	perGib, _ := big.NewFloat(0).SetInt(p.Price.PricePerGiB).Float64()
	perHour, _ := big.NewFloat(0).SetInt(p.Price.PricePerHour).Float64()
	prop := &proposalDTO{
		ProviderID:   p.ProviderID,
		ServiceType:  p.ServiceType,
		QualityLevel: proposalQualityLevelUnknown,
		Price: proposalPrice{
			Currency: money.CurrencyMyst.String(),
			PerGiB:   perGib,
			PerHour:  perHour,
		},
	}

	prop.Country = p.Location.Country
	prop.IPType = p.Location.IPType
	prop.QualityLevel = m.calculateMetricQualityLevel(p.Quality.Quality)

	return prop
}

func (m *proposalsManager) calculateMetricQualityLevel(quality float64) proposalQualityLevel {
	if quality == 0 {
		return proposalQualityLevelUnknown
	}

	if quality >= qualityLevelHigh {
		return proposalQualityLevelHigh
	}

	if quality >= qualityLevelMedium {
		return proposalQualityLevelMedium
	}

	return proposalQualityLevelLow
}
