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
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/market"
)

var mockProposal = market.ServiceProposal{
	ID:            1,
	Format:        "good format, i like it",
	Compatibility: 2,
	ProviderID:    "0x0",
	ServiceType:   "much service",
	Location: market.Location{
		IPType:    "residential",
		Country:   "yes",
		Continent: "no",
		ASN:       1,
		ISP:       "some isp",
	},
}

var presetRepository = &mockFilterPresetRepository{
	presets: proposal.FilterPresets{Entries: []proposal.FilterPreset{
		{
			ID:     0,
			Name:   "",
			IPType: "",
		},
	}},
}

func TestGetProposal(t *testing.T) {
	t.Run("fetches proposal correctly", func(t *testing.T) {
		mockPrice := market.Price{
			PricePerHour: big.NewInt(1),
			PricePerGiB:  big.NewInt(2),
		}
		mp := &mockPriceInfoProvider{
			priceToReturn: mockPrice,
		}

		mr := &mockRepository{
			proposalToReturn: &mockProposal,
			errToReturn:      nil,
		}

		repo := NewPricedServiceProposalRepository(mr, mp, presetRepository)

		result, err := repo.Proposal(market.ProposalID{})
		assert.NoError(t, err)

		assert.EqualValues(t, mockProposal, result.ServiceProposal)
		assert.EqualValues(t, mockPrice, result.Price)
	})
	t.Run("bubbles repo errors", func(t *testing.T) {
		mockError := errors.New("boom")
		mr := &mockRepository{
			errToReturn: mockError,
		}

		repo := NewPricedServiceProposalRepository(mr, &mockPriceInfoProvider{}, presetRepository)
		_, err := repo.Proposal(market.ProposalID{})
		assert.Error(t, err)
		assert.Equal(t, mockError, err)
	})
	t.Run("bubbles price errors", func(t *testing.T) {
		mockError := errors.New("boom")
		mp := &mockPriceInfoProvider{
			errorToReturn: mockError,
		}
		repo := NewPricedServiceProposalRepository(&mockRepository{
			proposalToReturn: &mockProposal,
		}, mp, nil)

		_, err := repo.Proposal(market.ProposalID{})
		assert.Error(t, err)
		assert.Equal(t, mockError, err)
	})
}

func TestGetProposals(t *testing.T) {
	t.Run("fetches proposals correctly", func(t *testing.T) {
		mockPrice := market.Price{
			PricePerHour: big.NewInt(1),
			PricePerGiB:  big.NewInt(2),
		}
		mp := &mockPriceInfoProvider{
			priceToReturn: mockPrice,
		}

		mr := &mockRepository{
			proposalsToReturn: []market.ServiceProposal{mockProposal},
			errToReturn:       nil,
		}

		repo := NewPricedServiceProposalRepository(mr, mp, presetRepository)

		result, err := repo.Proposals(nil)
		assert.NoError(t, err)

		assert.EqualValues(t, mockProposal, result[0].ServiceProposal)
		assert.EqualValues(t, mockPrice, result[0].Price)
	})
	t.Run("bubbles repo errors", func(t *testing.T) {
		mockError := errors.New("boom")
		mr := &mockRepository{
			errToReturn: mockError,
		}

		repo := NewPricedServiceProposalRepository(mr, &mockPriceInfoProvider{}, presetRepository)
		_, err := repo.Proposals(nil)
		assert.Error(t, err)
		assert.Equal(t, mockError, err)
	})
	t.Run("skips if price errors", func(t *testing.T) {
		mockError := errors.New("boom")
		mp := &mockPriceInfoProvider{
			errorToReturn: mockError,
		}
		repo := NewPricedServiceProposalRepository(&mockRepository{
			proposalsToReturn: []market.ServiceProposal{mockProposal},
		}, mp, presetRepository)

		res, err := repo.Proposals(nil)
		assert.NoError(t, err)
		assert.Len(t, res, 0)
	})
}

type mockRepository struct {
	proposalsToReturn []market.ServiceProposal
	errToReturn       error
	proposalToReturn  *market.ServiceProposal
}

func (mr *mockRepository) Proposal(id market.ProposalID) (*market.ServiceProposal, error) {
	return mr.proposalToReturn, mr.errToReturn
}

func (mr *mockRepository) Proposals(filter *proposal.Filter) ([]market.ServiceProposal, error) {
	return mr.proposalsToReturn, mr.errToReturn
}

func (mr *mockRepository) Countries(filter *proposal.Filter) (map[string]int, error) {
	return nil, nil
}

type mockPriceInfoProvider struct {
	priceToReturn market.Price
	errorToReturn error
}

func (mpip *mockPriceInfoProvider) GetCurrentPrice(nodeType string, country string, serviceType string) (market.Price, error) {
	return mpip.priceToReturn, mpip.errorToReturn
}

type mockFilterPresetRepository struct {
	presets proposal.FilterPresets
}

func (m *mockFilterPresetRepository) List() (*proposal.FilterPresets, error) {
	return &m.presets, nil
}

func (m *mockFilterPresetRepository) Get(id int) (*proposal.FilterPreset, error) {
	for _, p := range m.presets.Entries {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, errors.New("preset not found")
}
