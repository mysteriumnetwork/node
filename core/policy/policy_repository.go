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
	"time"

	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// PolicyRepository represents async policy fetcher from TrustOracle
type PolicyRepository struct {
	client *requests.HTTPClient

	policyURL   string
	policyLock  sync.Mutex
	policyRules map[market.AccessPolicy]market.AccessPolicyRuleSet

	fetchInterval time.Duration
	fetchShutdown chan struct{}
}

// NewPolicyRepository create instance of policy repository
func NewPolicyRepository(client *requests.HTTPClient, policyURL string, interval time.Duration) *PolicyRepository {
	return &PolicyRepository{
		client:        client,
		policyURL:     policyURL,
		policyRules:   make(map[market.AccessPolicy]market.AccessPolicyRuleSet),
		fetchInterval: interval,
		fetchShutdown: make(chan struct{}),
	}
}

// Start begins fetching proposals to repository
func (pr *PolicyRepository) Start() {
	pr.fetchShutdown = make(chan struct{})
	go pr.fetchLoop()
}

// Stop ends fetching proposals to repository
func (pr *PolicyRepository) Stop() {
	pr.fetchShutdown <- struct{}{}
}

// Policy converts given value to valid policy rule
func (pr *PolicyRepository) Policy(policyID string) market.AccessPolicy {
	return market.AccessPolicy{
		ID:     policyID,
		Source: fmt.Sprintf("%v%v", pr.policyURL, policyID),
	}
}

// Policies converts given values to list of valid policies
func (pr *PolicyRepository) Policies(policyIDs []string) []market.AccessPolicy {
	policies := make([]market.AccessPolicy, len(policyIDs))
	for i, policyID := range policyIDs {
		policies[i] = pr.Policy(policyID)
	}
	return policies
}

// AddPolicies adds given policy to repository. Also syncs policy rules from TrustOracle
func (pr *PolicyRepository) AddPolicies(policies []market.AccessPolicy) error {
	pr.policyLock.Lock()
	policyRulesNew := pr.policyRules
	pr.policyLock.Unlock()

	for _, policy := range policies {
		policyRules, err := pr.fetchPolicyRules(policy)
		if err != nil {
			return errors.Wrap(err, "initial fetch failed")
		}

		policyRulesNew[policy] = policyRules
	}

	pr.policyLock.Lock()
	pr.policyRules = policyRulesNew
	pr.policyLock.Unlock()

	return nil
}

// RulesForPolicy gives rules of given policy
func (pr *PolicyRepository) RulesForPolicy(policy market.AccessPolicy) (market.AccessPolicyRuleSet, error) {
	pr.policyLock.Lock()
	defer pr.policyLock.Unlock()

	policyRules, exist := pr.policyRules[policy]
	if !exist {
		return policyRules, fmt.Errorf("unknown policy: %s", policy)
	}
	return policyRules, nil
}

// RulesForPolicies gives list of rules of given policies
func (pr *PolicyRepository) RulesForPolicies(policies []market.AccessPolicy) ([]market.AccessPolicyRuleSet, error) {
	pr.policyLock.Lock()
	defer pr.policyLock.Unlock()

	policiesRules := make([]market.AccessPolicyRuleSet, len(policies))
	for i, policy := range policies {
		policyRules, exist := pr.policyRules[policy]
		if !exist {
			return policiesRules, fmt.Errorf("unknown policy: %s", policy)
		}
		policiesRules[i] = policyRules
	}
	return policiesRules, nil
}

func (pr *PolicyRepository) fetchPolicyRules(policy market.AccessPolicy) (market.AccessPolicyRuleSet, error) {
	var policyRules market.AccessPolicyRuleSet

	req, err := requests.NewGetRequest(policy.Source, "", nil)
	if err != nil {
		return policyRules, errors.Wrap(err, "failed to create policy request")
	}

	err = pr.client.DoRequestAndParseResponse(req, &policyRules)
	if err != nil {
		return policyRules, errors.Wrapf(err, "failed fetch policy rule %s", policy)
	}
	return policyRules, nil
}

func (pr *PolicyRepository) fetchLoop() {
	for {
		select {
		case <-pr.fetchShutdown:
			break
		case <-time.After(pr.fetchInterval):
			pr.policyLock.Lock()
			policyRulesActive := pr.policyRules
			pr.policyLock.Unlock()

			for policy := range policyRulesActive {
				policyRules, err := pr.fetchPolicyRules(policy)
				if err != nil {
					log.Warn().Err(err).Msg("synchronise fetch failed")
				}

				pr.policyLock.Lock()
				pr.policyRules[policy] = policyRules
				pr.policyLock.Unlock()
			}
		}
	}
}
