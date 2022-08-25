/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package discovery

import (
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/market"
)

// PricedServiceProposalRepository enriches proposals with price data as pricing data is not available on raw proposals.
type PricedServiceProposalRepository struct {
	baseRepo      proposal.Repository
	pip           PriceInfoProvider
	filterPresets proposal.FilterPresetRepository
}

// PriceInfoProvider allows to fetch the current pricing for services.
type PriceInfoProvider interface {
	GetCurrentPrice(nodeType string, country string, serviceType string) (market.Price, error)
}

// NewPricedServiceProposalRepository returns a new instance of PricedServiceProposalRepository.
func NewPricedServiceProposalRepository(baseRepo proposal.Repository, pip PriceInfoProvider, filterPresets proposal.FilterPresetRepository) *PricedServiceProposalRepository {
	return &PricedServiceProposalRepository{
		baseRepo:      baseRepo,
		pip:           pip,
		filterPresets: filterPresets,
	}
}

// Proposal fetches the proposal from base repository and enriches it with pricing data.
func (pspr *PricedServiceProposalRepository) Proposal(id market.ProposalID) (*proposal.PricedServiceProposal, error) {
	prop, err := pspr.baseRepo.Proposal(id)
	if err != nil {
		return nil, err
	}

	// base repo can sometimes return nil proposals.
	if prop == nil {
		return nil, nil
	}

	priced, err := pspr.toPricedProposal(*prop)
	return &priced, err
}

// Proposals fetches proposals from base repository and enriches them with pricing data.
func (pspr *PricedServiceProposalRepository) Proposals(filter *proposal.Filter) ([]proposal.PricedServiceProposal, error) {
	proposals, err := pspr.baseRepo.Proposals(filter)
	if err != nil {
		return nil, err
	}
	priced := pspr.toPricedProposals(proposals)

	if filter != nil && filter.PresetID != 0 {
		preset, err := pspr.filterPresets.Get(filter.PresetID)
		if err != nil {
			return nil, err
		}
		priced = preset.Filter(priced)
	}

	return priced, nil
}

// Countries fetches number of proposals per country from base repository.
func (pspr *PricedServiceProposalRepository) Countries(filter *proposal.Filter) (map[string]int, error) {
	return pspr.baseRepo.Countries(filter)
}

// EnrichProposalWithPrice adds pricing info to service proposal.
func (pspr *PricedServiceProposalRepository) EnrichProposalWithPrice(in market.ServiceProposal) (proposal.PricedServiceProposal, error) {
	return pspr.toPricedProposal(in)
}

func (pspr *PricedServiceProposalRepository) toPricedProposals(in []market.ServiceProposal) []proposal.PricedServiceProposal {
	res := make([]proposal.PricedServiceProposal, 0)
	for i := range in {
		priced, err := pspr.toPricedProposal(in[i])
		if err != nil {
			log.Warn().Err(err).Msgf("could not add pricing info to proposal %v(%v)", in[i].ProviderID, in[i].ServiceType)
			continue
		}
		res = append(res, priced)
	}

	return res
}

func (pspr *PricedServiceProposalRepository) toPricedProposal(in market.ServiceProposal) (proposal.PricedServiceProposal, error) {
	price, err := pspr.pip.GetCurrentPrice(in.Location.IPType, in.Location.Country, in.ServiceType)
	if err != nil {
		return proposal.PricedServiceProposal{}, err
	}

	return proposal.PricedServiceProposal{
		ServiceProposal: in,
		Price:           price,
	}, nil
}
