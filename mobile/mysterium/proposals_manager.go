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
	"fmt"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/mysterium"
	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/services/wireguard"
)

const (
	qualityLevelMedium = 1
	qualityLevelHigh   = 2
)

type proposalQualityLevel int

const (
	proposalQualityLevelUnknown proposalQualityLevel = 0
	proposalQualityLevelLow     proposalQualityLevel = 1
	proposalQualityLevelMedium  proposalQualityLevel = 2
	proposalQualityLevelHigh    proposalQualityLevel = 3
)

// GetProposalsRequest represents proposals request.
type GetProposalsRequest struct {
	ServiceType     string
	LocationCountry string
	IPType          string
	Refresh         bool
	PriceHourMax    float64
	PriceGiBMax     float64
	QualityMin      float32
	PresetID        int
}

func (r GetProposalsRequest) toFilter() *proposal.Filter {
	return &proposal.Filter{
		ServiceType:        r.ServiceType,
		LocationCountry:    r.LocationCountry,
		IPType:             r.IPType,
		QualityMin:         r.QualityMin,
		ExcludeUnsupported: true,
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
}

type getProposalsResponse struct {
	Proposals []*proposalDTO `json:"proposals"`
}

type getProposalResponse struct {
	Proposal *proposalDTO `json:"proposal"`
}

type mysteriumAPI interface {
	QueryProposals(query mysterium.ProposalsQuery) ([]market.ServiceProposal, error)
}

type qualityFinder interface {
	ProposalsQuality() []quality.ProposalQuality
}

func newProposalsManager(
	repository proposal.Repository,
	mysteriumAPI mysteriumAPI,
	filterPresetStorage *proposal.FilterPresetStorage,
) *proposalsManager {
	return &proposalsManager{
		repository:          repository,
		mysteriumAPI:        mysteriumAPI,
		filterPresetStorage: filterPresetStorage,
	}
}

type proposalsManager struct {
	repository          proposal.Repository
	cache               []market.ServiceProposal
	mysteriumAPI        mysteriumAPI
	filterPresetStorage *proposal.FilterPresetStorage
}

func (m *proposalsManager) getProposals(req *GetProposalsRequest) (*getProposalsResponse, error) {
	// Get proposals from cache if exists.
	if !req.Refresh {
		cachedProposals := m.getFromCache()
		if len(cachedProposals) > 0 {
			return m.mapToProposalsResponse(cachedProposals)
		}
	}

	apiProposals, err := m.getFromRepository(req.toFilter())
	if err != nil {
		return nil, err
	}

	if req.PresetID != 0 {
		preset, err := m.filterPresetStorage.Get(req.PresetID)
		if err != nil {
			return nil, err
		}
		apiProposals = preset.Filter(apiProposals)
	}

	m.addToCache(apiProposals)

	return m.mapToProposalsResponse(apiProposals)
}

func (m *proposalsManager) getFromCache() []market.ServiceProposal {
	return m.cache
}

func (m *proposalsManager) getFromRepository(filter *proposal.Filter) ([]market.ServiceProposal, error) {
	allProposals, err := m.repository.Proposals(filter)
	if err != nil {
		return nil, fmt.Errorf("could not get proposals from repository: %w", err)
	}

	// Ideally api should allow to pass multiple service types to skip noop
	// proposals, but for now just filter in memory.
	var res []market.ServiceProposal
	for _, p := range allProposals {
		if p.ServiceType == openvpn.ServiceType || p.ServiceType == wireguard.ServiceType {
			res = append(res, p)
		}
	}
	return res, nil
}

func (m *proposalsManager) addToCache(proposals []market.ServiceProposal) {
	m.cache = proposals
}

func (m *proposalsManager) mapToProposalsResponse(serviceProposals []market.ServiceProposal) (*getProposalsResponse, error) {
	var proposals []*proposalDTO
	for _, p := range serviceProposals {
		proposals = append(proposals, m.mapProposal(&p))
	}
	return &getProposalsResponse{Proposals: proposals}, nil
}

func (m *proposalsManager) mapProposal(p *market.ServiceProposal) *proposalDTO {
	prop := &proposalDTO{
		ProviderID:   p.ProviderID,
		ServiceType:  p.ServiceType,
		QualityLevel: proposalQualityLevelUnknown,
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
