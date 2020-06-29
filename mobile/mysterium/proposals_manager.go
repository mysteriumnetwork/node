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
	"fmt"

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
	ServiceType         string
	Refresh             bool
	IncludeFailed       bool
	UpperTimePriceBound int64
	LowerTimePriceBound int64
	UpperGBPriceBound   int64
	LowerGBPriceBound   int64
}

// GetProposalRequest represents proposal request.
type GetProposalRequest struct {
	ProviderID  string
	ServiceType string
}

type proposalDTO struct {
	ID               int                    `json:"id"`
	ProviderID       string                 `json:"providerId"`
	ServiceType      string                 `json:"serviceType"`
	CountryCode      string                 `json:"countryCode"`
	NodeType         string                 `json:"nodeType"`
	QualityLevel     proposalQualityLevel   `json:"qualityLevel"`
	MonitoringFailed bool                   `json:"monitoringFailed"`
	Payment          *proposalPaymentMethod `json:"payment"`
}

type proposalPaymentMethod struct {
	Type  string                `json:"type"`
	Price *proposalPaymentPrice `json:"price"`
	Rate  *proposalPaymentRate  `json:"rate"`
}

type proposalPaymentPrice struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type proposalPaymentRate struct {
	PerSeconds int64 `json:"perSeconds"`
	PerBytes   int64 `json:"perBytes"`
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
) *proposalsManager {
	return &proposalsManager{
		repository:    repository,
		mysteriumAPI:  mysteriumAPI,
		qualityFinder: qualityFinder,
	}
}

type proposalsManager struct {
	repository    proposal.Repository
	cache         []market.ServiceProposal
	mysteriumAPI  mysteriumAPI
	qualityFinder qualityFinder
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
	upperTimePriceBound := uint64(req.UpperTimePriceBound)
	lowerTimePriceBound := uint64(req.LowerTimePriceBound)
	upperGBPriceBound := uint64(req.UpperGBPriceBound)
	lowerGBPriceBound := uint64(req.LowerGBPriceBound)

	filter := &proposal.Filter{
		ServiceType:         req.ServiceType,
		UpperTimePriceBound: &upperTimePriceBound,
		LowerTimePriceBound: &lowerTimePriceBound,
		UpperGBPriceBound:   &upperGBPriceBound,
		LowerGBPriceBound:   &lowerGBPriceBound,
		ExcludeUnsupported:  true,
		IncludeFailed:       req.IncludeFailed,
	}
	apiProposals, err := m.getFromRepository(filter)
	if err != nil {
		return nil, err
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
	metrics := m.qualityFinder.ProposalsMetrics()
	metricsMap := map[string]quality.ConnectMetric{}
	for _, m := range metrics {
		metricsMap[m.ProposalID.ProviderID+m.ProposalID.ServiceType] = m
	}

	var proposals []*proposalDTO
	for _, p := range serviceProposals {
		proposals = append(proposals, m.mapProposal(&p, metricsMap))
	}

	res := &getProposalsResponse{Proposals: proposals}
	bytes, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (m *proposalsManager) mapProposal(p *market.ServiceProposal, metricsMap map[string]quality.ConnectMetric) *proposalDTO {
	prop := &proposalDTO{
		ID:           p.ID,
		ProviderID:   p.ProviderID,
		ServiceType:  p.ServiceType,
		QualityLevel: proposalQualityLevelUnknown,
	}

	if p.ServiceDefinition != nil {
		loc := p.ServiceDefinition.GetLocation()
		prop.CountryCode = loc.Country
		prop.NodeType = loc.NodeType
	}

	payment := p.PaymentMethod
	if payment != nil {
		prop.Payment = &proposalPaymentMethod{
			Type: payment.GetType(),
			Price: &proposalPaymentPrice{
				Amount:   int64(payment.GetPrice().Amount),
				Currency: string(payment.GetPrice().Currency),
			},
			Rate: &proposalPaymentRate{
				PerSeconds: int64(payment.GetRate().PerTime.Seconds()),
				PerBytes:   int64(payment.GetRate().PerByte),
			},
		}
	}

	if mc, ok := metricsMap[p.ProviderID+p.ServiceType]; ok {
		prop.QualityLevel = m.calculateMetricQualityLevel(mc.ConnectCount)
		prop.MonitoringFailed = mc.MonitoringFailed
	}

	return prop
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
