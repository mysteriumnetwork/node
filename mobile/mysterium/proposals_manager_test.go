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
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/mysterium"
	"github.com/mysteriumnetwork/node/money"
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
		nil,
	)
}

func (s *proposalManagerTestSuite) TestGetProposalsFromCache() {
	s.proposalsManager.cache = []market.ServiceProposal{
		{
			ProviderID:        "p1",
			ServiceType:       "openvpn",
			ServiceDefinition: &mockServiceDefinition{country: "usa", nodeType: "residential"},
			PaymentMethod:     &mockPayment{},
		},
	}
	s.proposalsManager.qualityFinder = &mockQualityFinder{
		quality: []quality.ProposalQuality{
			{
				ProposalID: quality.ProposalID{
					ProviderID:  "p1",
					ServiceType: "openvpn",
				},
				Quality: 2,
			},
		},
	}

	proposals, err := s.proposalsManager.getProposals(&GetProposalsRequest{
		Refresh:             false,
		UpperTimePriceBound: 0.005,
		LowerTimePriceBound: 0,
		UpperGBPriceBound:   0.7,
		LowerGBPriceBound:   0,
	})
	assert.NoError(s.T(), err)

	bytes, err := json.Marshal(&proposals)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "{\"proposals\":[{\"id\":0,\"providerId\":\"p1\",\"serviceType\":\"openvpn\",\"countryCode\":\"usa\",\"nodeType\":\"residential\",\"qualityLevel\":3,\"monitoringFailed\":false,\"payment\":{\"type\":\"pt\",\"price\":{\"amount\":1e-17,\"currency\":\"MYSTT\"},\"rate\":{\"perSeconds\":10,\"perBytes\":15}}}]}", string(bytes))
}

func (s *proposalManagerTestSuite) TestGetProposalsFromAPIWhenNotFoundInCache() {
	s.repository.data = []market.ServiceProposal{
		{
			ProviderID:        "p1",
			ServiceType:       "wireguard",
			ServiceDefinition: mockServiceDefinition{country: "usa", nodeType: "residential"},
			PaymentMethod:     &mockPayment{},
		},
	}
	s.proposalsManager.mysteriumAPI = &mockMysteriumAPI{}
	proposals, err := s.proposalsManager.getProposals(&GetProposalsRequest{
		Refresh: true,
	})
	assert.NoError(s.T(), err)

	bytes, err := json.Marshal(&proposals)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "{\"proposals\":[{\"id\":0,\"providerId\":\"p1\",\"serviceType\":\"wireguard\",\"countryCode\":\"usa\",\"nodeType\":\"residential\",\"qualityLevel\":0,\"monitoringFailed\":false,\"payment\":{\"type\":\"pt\",\"price\":{\"amount\":1e-17,\"currency\":\"MYSTT\"},\"rate\":{\"perSeconds\":10,\"perBytes\":15}}}]}", string(bytes))
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

type mockServiceDefinition struct {
	country  string
	nodeType string
}

func (m mockServiceDefinition) GetLocation() market.Location {
	return market.Location{
		Continent: "",
		Country:   m.country,
		City:      "",
		ASN:       0,
		ISP:       "",
		NodeType:  m.nodeType,
	}
}

type mockPayment struct{}

func (m mockPayment) GetType() string {
	return "pt"
}

func (m mockPayment) GetPrice() money.Money {
	return money.Money{
		Amount:   big.NewInt(10),
		Currency: "MYSTT",
	}
}

func (m mockPayment) GetRate() market.PaymentRate {
	return market.PaymentRate{
		PerTime: 10 * time.Second,
		PerByte: 15,
	}
}
