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
	"testing"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/mysterium"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type proposalManagerTestSuite struct {
	suite.Suite

	repository    *mockRepository
	mysteriumAPI  mysteriumAPI
	qualityFinder qualityFinder

	proposalsManager *proposalsManager
}

func (s *proposalManagerTestSuite) SetupTest() {
	s.repository = &mockRepository{}
	s.mysteriumAPI = &mockMysteriumAPI{}
	s.qualityFinder = &mockQualityFinder{}

	s.proposalsManager = newProposalsManager(
		s.repository,
		s.mysteriumAPI,
		s.qualityFinder,
		0, 1000000,
	)
}

func (s *proposalManagerTestSuite) TestGetProposalsFromCache() {
	s.proposalsManager.cache = []market.ServiceProposal{
		{ProviderID: "p1", ServiceType: "openvpn"},
	}
	s.proposalsManager.qualityFinder = &mockQualityFinder{
		metrics: []quality.ConnectMetric{
			{
				ProposalID: quality.ProposalID{
					ProviderID:  "p1",
					ServiceType: "openvpn",
				},
				ConnectCount: quality.ConnectCount{
					Success: 23,
					Fail:    4,
					Timeout: 6,
				},
			},
		},
	}

	bytes, err := s.proposalsManager.getProposals(&GetProposalsRequest{
		ShowOpenvpnProposals:   false,
		ShowWireguardProposals: false,
		Refresh:                false,
	})

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "{\"proposals\":[{\"id\":0,\"providerId\":\"p1\",\"serviceType\":\"openvpn\",\"countryCode\":\"\",\"qualityLevel\":3}]}", string(bytes))
}

func (s *proposalManagerTestSuite) TestGetProposalsFromAPIWhenNotFoundInCache() {
	s.repository.data = []market.ServiceProposal{
		{ProviderID: "p1", ServiceType: "wireguard"},
	}
	s.proposalsManager.mysteriumAPI = &mockMysteriumAPI{}
	bytes, err := s.proposalsManager.getProposals(&GetProposalsRequest{
		ShowOpenvpnProposals:   false,
		ShowWireguardProposals: false,
		Refresh:                true,
	})

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "{\"proposals\":[{\"id\":0,\"providerId\":\"p1\",\"serviceType\":\"wireguard\",\"countryCode\":\"\",\"qualityLevel\":0}]}", string(bytes))
}

func (s *proposalManagerTestSuite) TestGetSingleProposal() {
	s.repository.data = []market.ServiceProposal{
		{ProviderID: "p1", ServiceType: "wireguard"},
	}
	bytes, err := s.proposalsManager.getProposal(&GetProposalRequest{
		ProviderID:  "p1",
		ServiceType: "wireguard",
	})

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "{\"proposal\":{\"id\":0,\"providerId\":\"p1\",\"serviceType\":\"wireguard\",\"countryCode\":\"\",\"qualityLevel\":0}}", string(bytes))
}

func TestProposalManagerSuite(t *testing.T) {
	suite.Run(t, new(proposalManagerTestSuite))
}

type mockRepository struct {
	data []market.ServiceProposal
}

func (m *mockRepository) Proposal(id market.ProposalID) (*market.ServiceProposal, error) {
	if len(m.data) == 0 {
		return nil, nil
	}
	return &m.data[0], nil
}

func (m *mockRepository) Proposals(filter *proposal.Filter) ([]market.ServiceProposal, error) {
	return m.data, nil
}

type mockMysteriumAPI struct {
	proposals []market.ServiceProposal
}

func (m *mockMysteriumAPI) QueryProposals(query mysterium.ProposalsQuery) ([]market.ServiceProposal, error) {
	return m.proposals, nil
}

type mockQualityFinder struct {
	metrics []quality.ConnectMetric
}

func (m *mockQualityFinder) ProposalsMetrics() []quality.ConnectMetric {
	return m.metrics
}
