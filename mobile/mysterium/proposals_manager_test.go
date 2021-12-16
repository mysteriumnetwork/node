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
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat"
)

type proposalManagerTestSuite struct {
	suite.Suite

	repository       *mockRepository
	proposalsManager *proposalsManager
}

func (s *proposalManagerTestSuite) SetupTest() {
	s.repository = &mockRepository{}

	s.proposalsManager = newProposalsManager(
		s.repository,
		nil,
		&mockNATProber{"none", nil},
		60*time.Second,
	)
}

func (s *proposalManagerTestSuite) TestGetProposalsFromCache() {
	s.proposalsManager.cachedAt = time.Now().Add(1 * time.Hour)
	s.proposalsManager.cache = []proposal.PricedServiceProposal{
		{
			ServiceProposal: market.NewProposal("p1", "openvpn", market.NewProposalOpts{
				Location: &market.Location{
					Country: "US",
					IPType:  "residential",
				},
				Quality: &market.Quality{Quality: 2, Latency: 50, Bandwidth: 10},
			}),
			Price: market.Price{
				PricePerHour: big.NewInt(1),
				PricePerGiB:  big.NewInt(2),
			},
		},
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
		  "quality_level": 3,
		  "price": {
			  "per_gib": 2.0,
			  "per_hour": 1.0,
			  "currency": "MYST"
		  }
		}
	  ]
	}`, string(bytes))
}

func (s *proposalManagerTestSuite) TestGetProposalsFromAPIWhenNotFoundInCache() {
	s.repository.data = []proposal.PricedServiceProposal{
		{
			ServiceProposal: market.NewProposal("p1", "wireguard", market.NewProposalOpts{
				Location: &market.Location{
					Country: "US",
					IPType:  "residential",
				},
				Quality: &market.Quality{Quality: 2, Latency: 50, Bandwidth: 10},
			}),
			Price: market.Price{
				PricePerHour: big.NewInt(1),
				PricePerGiB:  big.NewInt(2),
			},
		},
	}
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
		  "quality_level": 3,
		  "price": {
			"per_gib": 2.0,
			"per_hour": 1.0,
			"currency": "MYST"
		  }
		}
	  ]
	}`, string(bytes))
}

func TestProposalManagerSuite(t *testing.T) {
	suite.Run(t, new(proposalManagerTestSuite))
}

type mockRepository struct {
	data []proposal.PricedServiceProposal
}

func (m *mockRepository) Proposal(id market.ProposalID) (*proposal.PricedServiceProposal, error) {
	if len(m.data) == 0 {
		return nil, nil
	}
	return &m.data[0], nil
}

func (m *mockRepository) Proposals(filter *proposal.Filter) ([]proposal.PricedServiceProposal, error) {
	return m.data, nil
}

func (m *mockRepository) Countries(filter *proposal.Filter) (map[string]int, error) {
	return nil, nil
}

type mockNATProber struct {
	returnRes nat.NATType
	returnErr error
}

func (m *mockNATProber) Probe(_ context.Context) (nat.NATType, error) {
	return m.returnRes, m.returnErr
}
