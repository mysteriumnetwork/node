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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/mysterium"
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
		nil,
	)
}

func (s *proposalManagerTestSuite) TestGetProposalsFromCache() {
	s.proposalsManager.cache = []market.ServiceProposal{
		market.NewProposal("p1", "openvpn", market.NewProposalOpts{
			Location: &market.Location{
				Country: "US",
				IPType:  "residential",
			},
			Quality: &market.Quality{Quality: 2, Latency: 50, Bandwidth: 10},
		}),
	}

	proposals, err := s.proposalsManager.getProposals(&GetProposalsRequest{
		Refresh:      false,
		PriceHourMax: 0.005,
		PriceGiBMax:  0.7,
	})
	assert.NoError(s.T(), err)

	bytes, err := json.Marshal(&proposals)
	assert.NoError(s.T(), err)
	assert.JSONEq(s.T(), `{
	  "proposals": [
		{
		  "provider_id": "p1",
		  "service_type": "openvpn",
		  "country": "US",
		  "ip_type": "residential",
		  "quality_level": 3
		}
	  ]
	}`, string(bytes))
}

func (s *proposalManagerTestSuite) TestGetProposalsFromAPIWhenNotFoundInCache() {
	s.repository.data = []market.ServiceProposal{
		market.NewProposal("p1", "wireguard", market.NewProposalOpts{
			Location: &market.Location{
				Country: "US",
				IPType:  "residential",
			},
			Quality: &market.Quality{Quality: 2, Latency: 50, Bandwidth: 10},
		}),
	}
	s.proposalsManager.mysteriumAPI = &mockMysteriumAPI{}
	proposals, err := s.proposalsManager.getProposals(&GetProposalsRequest{
		Refresh: true,
	})
	assert.NoError(s.T(), err)

	bytes, err := json.Marshal(&proposals)
	assert.NoError(s.T(), err)
	assert.JSONEq(s.T(), `{
	  "proposals": [
		{
		  "provider_id": "p1",
		  "service_type": "wireguard",
		  "country": "US",
		  "ip_type": "residential",
		  "quality_level": 3
		}
	  ]
	}`, string(bytes))
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
	quality []quality.ProposalQuality
}

func (m *mockQualityFinder) ProposalsQuality() []quality.ProposalQuality {
	return m.quality
}
