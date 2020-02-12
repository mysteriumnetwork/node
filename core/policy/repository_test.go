/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package policy

import (
	"testing"

	"github.com/mysteriumnetwork/node/market"
	"github.com/stretchr/testify/assert"
)

var (
	policyOne      = market.AccessPolicy{ID: "1", Source: "http://policy.localhost/1"}
	policyOneRules = market.AccessPolicyRuleSet{
		ID:    "1",
		Title: "One",
		Allow: []market.AccessRule{
			{Type: market.AccessPolicyTypeIdentity, Value: "0x1"},
		},
	}
	policyOneRulesUpdated = market.AccessPolicyRuleSet{
		ID:    "1",
		Title: "One (updated)",
		Allow: []market.AccessRule{
			{Type: market.AccessPolicyTypeIdentity, Value: "0x1"},
		},
	}

	policyTwo      = market.AccessPolicy{ID: "2", Source: "http://policy.localhost/2"}
	policyTwoRules = market.AccessPolicyRuleSet{
		ID:    "2",
		Title: "Two",
		Allow: []market.AccessRule{
			{Type: market.AccessPolicyTypeDNSHostname, Value: "ipinfo.io"},
		},
	}
	policyTwoRulesUpdated = market.AccessPolicyRuleSet{
		ID:    "2",
		Title: "Two (updated)",
		Allow: []market.AccessRule{
			{Type: market.AccessPolicyTypeDNSHostname, Value: "ipinfo.io"},
		},
	}

	policyThree             = market.AccessPolicy{ID: "3", Source: "http://policy.localhost/3"}
	policyThreeRulesUpdated = market.AccessPolicyRuleSet{
		ID:    "3",
		Title: "Three (updated)",
		Allow: []market.AccessRule{
			{Type: market.AccessPolicyTypeDNSZone, Value: "ipinfo.io"},
		},
	}
)

func Test_Repository_RulesForPolicy(t *testing.T) {
	repo := createEmptyRepo()
	policyRules, err := repo.RulesForPolicy(policyOne)
	assert.EqualError(t, err, "unknown policy: {1 http://policy.localhost/1}")
	assert.Equal(t, market.AccessPolicyRuleSet{}, policyRules)

	repo = createFullRepo()
	policyRules, err = repo.RulesForPolicy(policyOne)
	assert.NoError(t, err)
	assert.Equal(t, policyOneRules, policyRules)
}

func Test_Repository_RulesForPolicies(t *testing.T) {
	repo := createEmptyRepo()
	policiesRules, err := repo.RulesForPolicies([]market.AccessPolicy{
		policyOne,
		policyThree,
	})
	assert.EqualError(t, err, "unknown policy: {1 http://policy.localhost/1}")
	assert.Equal(t, []market.AccessPolicyRuleSet{}, policiesRules)

	repo = createFullRepo()
	policiesRules, err = repo.RulesForPolicies([]market.AccessPolicy{
		policyOne,
		policyTwo,
	})
	assert.NoError(t, err)
	assert.Equal(t, []market.AccessPolicyRuleSet{policyOneRules, policyTwoRules}, policiesRules)
}

func Test_Repository_Rules(t *testing.T) {
	repo := createEmptyRepo()
	assert.Equal(t, []market.AccessPolicyRuleSet{}, repo.Rules())

	repo = createFullRepo()
	assert.Equal(t, []market.AccessPolicyRuleSet{policyOneRules, policyTwoRules}, repo.Rules())
}

func createEmptyRepo() *Repository {
	return NewRepository()
}

func createFullRepo() *Repository {
	repo := NewRepository()
	repo.SetPolicyRules(
		policyOne,
		market.AccessPolicyRuleSet{
			ID:    "1",
			Title: "One",
			Allow: []market.AccessRule{
				{Type: market.AccessPolicyTypeIdentity, Value: "0x1"},
			},
		},
	)
	repo.SetPolicyRules(
		policyTwo,
		market.AccessPolicyRuleSet{
			ID:    "2",
			Title: "Two",
			Allow: []market.AccessRule{
				{Type: market.AccessPolicyTypeDNSHostname, Value: "ipinfo.io"},
			},
		},
	)
	return repo
}
