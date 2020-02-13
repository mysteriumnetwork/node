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
	"strings"
	"sync"

	"github.com/mysteriumnetwork/node/market"
)

type listItem struct {
	policy market.AccessPolicy
	rules  market.AccessPolicyRuleSet
}

// Repository represents async policy fetcher from TrustOracle
type Repository struct {
	lock  sync.RWMutex
	items []listItem
}

// NewRepository create instance of policy repository
func NewRepository() *Repository {
	return &Repository{
		items: make([]listItem, 0),
	}
}

// SetPolicyRules set policy ant it's items to repository
func (r *Repository) SetPolicyRules(policy market.AccessPolicy, policyRules market.AccessPolicyRuleSet) {
	r.lock.Lock()
	defer r.lock.Unlock()

	item, err := r.findItemFor(policy)
	if err != nil {
		r.items = append(r.items, listItem{
			policy: policy,
			rules:  policyRules,
		})
	} else {
		item.rules = policyRules
	}
}

// Policies list policies in repository
func (r *Repository) Policies() []market.AccessPolicy {
	r.lock.RLock()
	defer r.lock.RUnlock()

	policies := make([]market.AccessPolicy, 0)
	for _, item := range r.items {
		policies = append(policies, item.policy)
	}

	return policies
}

// RulesForPolicy gives items of given polic
func (r *Repository) RulesForPolicy(policy market.AccessPolicy) (market.AccessPolicyRuleSet, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	item, err := r.findItemFor(policy)
	if err != nil {
		return market.AccessPolicyRuleSet{}, err
	}

	return item.rules, nil
}

// RulesForPolicies gives list of items of given policies
func (r *Repository) RulesForPolicies(policies []market.AccessPolicy) ([]market.AccessPolicyRuleSet, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	policiesRules := make([]market.AccessPolicyRuleSet, len(policies))
	for i, policy := range policies {
		item, err := r.findItemFor(policy)
		if err != nil {
			return []market.AccessPolicyRuleSet{}, fmt.Errorf("unknown policy: %s", policy)
		}
		policiesRules[i] = item.rules
	}

	return policiesRules, nil
}

// Rules gives list of items all policies
func (r *Repository) Rules() []market.AccessPolicyRuleSet {
	r.lock.RLock()
	defer r.lock.RUnlock()

	policiesRules := make([]market.AccessPolicyRuleSet, 0)
	for _, item := range r.items {
		policiesRules = append(policiesRules, item.rules)
	}

	return policiesRules
}

// HasDNSRules returns flag if any DNS rules are applied
func (r *Repository) HasDNSRules() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()

	for _, item := range r.items {
		for _, rule := range item.rules.Allow {
			if rule.Type == market.AccessPolicyTypeDNSZone {
				return true
			}
			if rule.Type == market.AccessPolicyTypeDNSHostname {
				return true
			}
		}
	}

	return false
}

// IsHostAllowed returns flag if given FQDN host should be allowed by rules
func (r *Repository) IsHostAllowed(host string) bool {
	r.lock.RLock()
	defer r.lock.RUnlock()

	hasDNSRules := false
	for _, item := range r.items {
		for _, rule := range item.rules.Allow {
			if rule.Type == market.AccessPolicyTypeDNSZone {
				hasDNSRules = true
				if strings.HasSuffix(host, rule.Value) {
					return true
				}
			}
			if rule.Type == market.AccessPolicyTypeDNSHostname {
				hasDNSRules = true
				if host == rule.Value {
					return true
				}
			}
		}
	}

	if !hasDNSRules {
		return true
	}
	return false
}

func (r *Repository) findItemFor(policy market.AccessPolicy) (*listItem, error) {
	for _, item := range r.items {
		if item.policy == policy {
			return &item, nil
		}
	}
	return nil, fmt.Errorf("unknown policy: %s", policy)
}
