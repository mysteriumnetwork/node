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

package policy

import (
	"fmt"
	"sync"

	"github.com/mysteriumnetwork/node/market"
)

// Repository represents async policy fetcher from TrustOracle
type Repository struct {
	lock          sync.RWMutex
	rulesByPolicy map[market.AccessPolicy]market.AccessPolicyRuleSet
}

// NewRepository create instance of policy repository
func NewRepository() *Repository {
	return &Repository{
		rulesByPolicy: make(map[market.AccessPolicy]market.AccessPolicyRuleSet),
	}
}

// SetPolicyRules set policy ant it's rules to repository
func (pr *Repository) SetPolicyRules(policy market.AccessPolicy, policyRules market.AccessPolicyRuleSet) {
	pr.lock.Lock()
	defer pr.lock.Unlock()

	pr.rulesByPolicy[policy] = policyRules
}

// Policies list policies in repository
func (pr *Repository) Policies() []market.AccessPolicy {
	pr.lock.RLock()
	defer pr.lock.RUnlock()

	policies := make([]market.AccessPolicy, len(pr.rulesByPolicy))
	i := 0
	for policy := range pr.rulesByPolicy {
		policies[i] = policy
		i++
	}

	return policies
}

// RulesForPolicy gives rules of given policy
func (pr *Repository) RulesForPolicy(policy market.AccessPolicy) (market.AccessPolicyRuleSet, error) {
	pr.lock.RLock()
	defer pr.lock.RUnlock()

	policyRules, exist := pr.rulesByPolicy[policy]
	if !exist {
		return market.AccessPolicyRuleSet{}, fmt.Errorf("unknown policy: %s", policy)
	}

	return policyRules, nil
}

// RulesForPolicies gives list of rules of given policies
func (pr *Repository) RulesForPolicies(policies []market.AccessPolicy) ([]market.AccessPolicyRuleSet, error) {
	pr.lock.RLock()
	defer pr.lock.RUnlock()

	policiesRules := make([]market.AccessPolicyRuleSet, len(policies))
	for i, policy := range policies {
		var exist bool
		policiesRules[i], exist = pr.rulesByPolicy[policy]
		if !exist {
			return []market.AccessPolicyRuleSet{}, fmt.Errorf("unknown policy: %s", policy)
		}
	}

	return policiesRules, nil
}
