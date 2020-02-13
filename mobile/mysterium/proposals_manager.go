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
	"encoding/json"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/mysterium"
	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/services/wireguard"
)

const (
	qualityLevelMedium = 0.2
	qualityLevelHigh   = 0.5
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
	ShowOpenvpnProposals   bool
	ShowWireguardProposals bool
	Refresh                bool
}

// GetProposalRequest represents proposal request.
type GetProposalRequest struct {
	ProviderID  string
	ServiceType string
}

type proposalDTO struct {
	ID           int                  `json:"id"`
	ProviderID   string               `json:"providerId"`
	ServiceType  string               `json:"serviceType"`
	CountryCode  string               `json:"countryCode"`
	QualityLevel proposalQualityLevel `json:"qualityLevel"`
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
	ProposalsMetrics() []quality.ConnectMetric
}

func newProposalsManager(
	repository proposal.Repository,
	mysteriumAPI mysteriumAPI,
	qualityFinder qualityFinder,
	lowerTimePriceBound, upperTimePriceBound uint64,
	lowerGBPriceBound, upperGBPriceBound uint64,

) *proposalsManager {
	return &proposalsManager{
		repository:          repository,
		mysteriumAPI:        mysteriumAPI,
		qualityFinder:       qualityFinder,
		upperTimePriceBound: upperTimePriceBound,
		lowerTimePriceBound: lowerTimePriceBound,
		lowerGBPriceBound:   lowerGBPriceBound,
		upperGBPriceBound:   upperGBPriceBound,
	}
}

type proposalsManager struct {
	repository          proposal.Repository
	cache               []market.ServiceProposal
	mysteriumAPI        mysteriumAPI
	qualityFinder       qualityFinder
	upperTimePriceBound uint64
	lowerTimePriceBound uint64
	lowerGBPriceBound   uint64
	upperGBPriceBound   uint64
}

func (m *proposalsManager) getProposals(req *GetProposalsRequest) ([]byte, error) {
	// Get proposals from cache if exists.
	if !req.Refresh {
		cachedProposals := m.getFromCache()
		if len(cachedProposals) > 0 {
			return m.mapToProposalsResponse(cachedProposals)
		}
	}

	// Get proposals from remote discovery api and store in cache.
	apiProposals, err := m.getFromRepository()
	if err != nil {
		return nil, err
	}
	m.addToCache(apiProposals)

	return m.mapToProposalsResponse(apiProposals)
}

func (m *proposalsManager) getProposal(req *GetProposalRequest) ([]byte, error) {
	result, err := m.repository.Proposal(market.ProposalID{
		ProviderID:  req.ProviderID,
		ServiceType: req.ServiceType,
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return m.mapToProposalResponse(result)
}

func (m *proposalsManager) getFromCache() []market.ServiceProposal {
	return m.cache
}

func (m *proposalsManager) getFromRepository() ([]market.ServiceProposal, error) {
	allProposals, err := m.repository.Proposals(&proposal.Filter{
		LowerGBPriceBound:   &m.lowerGBPriceBound,
		LowerTimePriceBound: &m.lowerTimePriceBound,
		UpperGBPriceBound:   &m.upperGBPriceBound,
		UpperTimePriceBound: &m.upperTimePriceBound,
	})
	if err != nil {
		return nil, err
	}

	// Ideally api should allow to pass multiple service types to skip noop
	// proposals, but for now jus filter in memory.
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

func (m *proposalsManager) mapToProposalsResponse(serviceProposals []market.ServiceProposal) ([]byte, error) {
	var proposals []*proposalDTO
	for _, p := range serviceProposals {
		proposals = append(proposals, &proposalDTO{
			ID:          p.ID,
			ProviderID:  p.ProviderID,
			ServiceType: p.ServiceType,
			CountryCode: m.getServiceCountryCode(&p),
		})
	}

	m.addQualityData(proposals)

	res := &getProposalsResponse{Proposals: proposals}
	bytes, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (m *proposalsManager) mapToProposalResponse(p *market.ServiceProposal) ([]byte, error) {
	dto := &proposalDTO{
		ID:          p.ID,
		ProviderID:  p.ProviderID,
		ServiceType: p.ServiceType,
		CountryCode: m.getServiceCountryCode(p),
	}
	res := &getProposalResponse{Proposal: dto}
	bytes, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (m *proposalsManager) getServiceCountryCode(p *market.ServiceProposal) string {
	if p.ServiceDefinition == nil {
		return ""
	}
	return p.ServiceDefinition.GetLocation().Country
}

func (m *proposalsManager) addQualityData(proposals []*proposalDTO) {
	metrics := m.qualityFinder.ProposalsMetrics()

	// Convert metrics slice to map for fast lookup.
	metricsMap := map[string]quality.ConnectMetric{}
	for _, m := range metrics {
		metricsMap[m.ProposalID.ProviderID+m.ProposalID.ServiceType] = m
	}

	for _, p := range proposals {
		p.QualityLevel = proposalQualityLevelUnknown
		if mc, ok := metricsMap[p.ProviderID+p.ServiceType]; ok {
			p.QualityLevel = m.calculateMetricQualityLevel(mc.ConnectCount)
		}
	}
}

func (m *proposalsManager) calculateMetricQualityLevel(counts quality.ConnectCount) proposalQualityLevel {
	total := counts.Success + counts.Fail + counts.Timeout
	if total == 0 {
		return proposalQualityLevelUnknown
	}

	qualityRatio := float64(counts.Success) / float64(total)
	if qualityRatio >= qualityLevelHigh {
		return proposalQualityLevelHigh
	}
	if qualityRatio >= qualityLevelMedium {
		return proposalQualityLevelMedium
	}
	return proposalQualityLevelLow
}
