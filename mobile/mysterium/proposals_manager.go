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

	"github.com/mysteriumnetwork/node/core/discovery"
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

type proposal struct {
	ID           int                  `json:"id"`
	ProviderID   string               `json:"providerId"`
	ServiceType  string               `json:"serviceType"`
	CountryCode  string               `json:"countryCode"`
	QualityLevel proposalQualityLevel `json:"qualityLevel"`
}

type getProposalsResponse struct {
	Proposals []*proposal `json:"proposals"`
}

type getProposalResponse struct {
	Proposal *proposal `json:"proposal"`
}

type discoveryFinder interface {
	GetProposal(id market.ProposalID) (*market.ServiceProposal, error)
	MatchProposals(match discovery.ProposalReducer) ([]market.ServiceProposal, error)
}

type proposalStorage interface {
	Set(proposals ...market.ServiceProposal)
}

type mysteriumAPI interface {
	QueryProposals(query mysterium.ProposalsQuery) ([]market.ServiceProposal, error)
}

type qualityFinder interface {
	ProposalsMetrics() []quality.ConnectMetric
}

func newProposalsManager(
	discoveryFinder discoveryFinder,
	proposalsStore proposalStorage,
	mysteriumAPI mysteriumAPI,
	qualityFinder qualityFinder,
) *proposalsManager {
	return &proposalsManager{
		discoveryFinder: discoveryFinder,
		proposalsStore:  proposalsStore,
		mysteriumAPI:    mysteriumAPI,
		qualityFinder:   qualityFinder,
	}
}

type proposalsManager struct {
	discoveryFinder discoveryFinder
	proposalsStore  proposalStorage
	mysteriumAPI    mysteriumAPI
	qualityFinder   qualityFinder
}

func (m *proposalsManager) getProposals(req *GetProposalsRequest) ([]byte, error) {
	// Get proposals from cache if exists.
	if !req.Refresh {
		cachedProposals, err := m.getFromCache()
		if err != nil {
			return nil, err
		}
		if len(cachedProposals) > 0 {
			return m.mapToProposalsResponse(cachedProposals)
		}
	}

	// Get proposals from remote discovery api and store in cache.
	apiProposals, err := m.getFromAPI(req.ShowOpenvpnProposals, req.ShowWireguardProposals)
	if err != nil {
		return nil, err
	}
	m.addToCache(apiProposals)

	return m.mapToProposalsResponse(apiProposals)
}

func (m *proposalsManager) getProposal(req *GetProposalRequest) ([]byte, error) {
	proposal, err := m.discoveryFinder.GetProposal(market.ProposalID{
		ProviderID:  req.ProviderID,
		ServiceType: req.ServiceType,
	})
	if err != nil {
		return nil, err
	}

	if proposal == nil {
		return nil, nil
	}
	return m.mapToProposalResponse(proposal)
}

func (m *proposalsManager) getFromCache() ([]market.ServiceProposal, error) {
	return m.discoveryFinder.MatchProposals(func(v market.ServiceProposal) bool {
		return true
	})
}

func (m *proposalsManager) getFromAPI(showOpenvpnProposals, showWireguardProposals bool) ([]market.ServiceProposal, error) {
	var serviceType string
	if showOpenvpnProposals && showWireguardProposals {
		serviceType = "all"
	} else if showOpenvpnProposals {
		serviceType = openvpn.ServiceType
	} else if showWireguardProposals {
		serviceType = wireguard.ServiceType
	}
	query := mysterium.ProposalsQuery{
		ServiceType:     serviceType,
		AccessPolicyAll: false,
	}
	allProposals, err := m.mysteriumAPI.QueryProposals(query)
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
	m.proposalsStore.Set(proposals...)
}

func (m *proposalsManager) mapToProposalsResponse(serviceProposals []market.ServiceProposal) ([]byte, error) {
	var proposals []*proposal
	for _, p := range serviceProposals {
		proposals = append(proposals, &proposal{
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
	proposal := &proposal{
		ID:          p.ID,
		ProviderID:  p.ProviderID,
		ServiceType: p.ServiceType,
		CountryCode: m.getServiceCountryCode(p),
	}
	res := &getProposalResponse{Proposal: proposal}
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

func (m *proposalsManager) addQualityData(proposals []*proposal) {
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
