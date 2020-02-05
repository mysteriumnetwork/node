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
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/logconfig/httptrace"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type policyMetadata struct {
	policy market.AccessPolicy
	eTag   string
	rules  market.AccessPolicyRuleSet
}

// Repository represents async policy fetcher from TrustOracle
type Repository struct {
	client *requests.HTTPClient

	policyURL  string
	policyLock sync.RWMutex
	policyList []policyMetadata

	fetchInterval time.Duration
	fetchShutdown chan struct{}
}

// NewRepository create instance of policy repository
func NewRepository(client *requests.HTTPClient, policyURL string, interval time.Duration) *Repository {
	return &Repository{
		client:        client,
		policyURL:     policyURL,
		policyList:    make([]policyMetadata, 0),
		fetchInterval: interval,
		fetchShutdown: make(chan struct{}),
	}
}

// Start begins fetching policies to repository
func (pr *Repository) Start() {
	go pr.fetchLoop()
}

// Stop ends fetching policies to repository
func (pr *Repository) Stop() {
	pr.fetchShutdown <- struct{}{}
}

// Policy converts given value to valid policy rule
func (pr *Repository) Policy(policyID string) market.AccessPolicy {
	policyURL := pr.policyURL
	if !strings.HasSuffix(policyURL, "/") {
		policyURL += "/"
	}
	return market.AccessPolicy{
		ID:     policyID,
		Source: fmt.Sprintf("%v%v", policyURL, policyID),
	}
}

// Policies converts given values to list of valid policies
func (pr *Repository) Policies(policyIDs []string) *[]market.AccessPolicy {
	policies := make([]market.AccessPolicy, len(policyIDs))
	for i, policyID := range policyIDs {
		policies[i] = pr.Policy(policyID)
	}
	return &policies
}

// AddPolicies adds given policy to repository. Also syncs policy rules from TrustOracle
func (pr *Repository) AddPolicies(policies []market.AccessPolicy) error {
	pr.policyLock.Lock()
	defer pr.policyLock.Unlock()

	policyListNew := make([]policyMetadata, len(pr.policyList))
	copy(policyListNew, pr.policyList)

	for _, policy := range policies {
		index, exist := pr.getPolicyIndex(policyListNew, policy)
		if !exist {
			index = len(policyListNew)
			policyListNew = append(policyListNew, policyMetadata{policy: policy})
		}

		var err error
		if err = pr.fetchPolicyRules(&policyListNew[index]); err != nil {
			return errors.Wrap(err, "initial fetch failed")
		}
	}

	pr.policyList = policyListNew
	return nil
}

// RulesForPolicy gives rules of given policy
func (pr *Repository) RulesForPolicy(policy market.AccessPolicy) (market.AccessPolicyRuleSet, error) {
	pr.policyLock.RLock()
	defer pr.policyLock.RUnlock()

	index, exist := pr.getPolicyIndex(pr.policyList, policy)
	if !exist {
		return market.AccessPolicyRuleSet{}, fmt.Errorf("unknown policy: %s", policy)
	}

	return pr.policyList[index].rules, nil
}

// RulesForPolicies gives list of rules of given policies
func (pr *Repository) RulesForPolicies(policies []market.AccessPolicy) ([]market.AccessPolicyRuleSet, error) {
	pr.policyLock.RLock()
	defer pr.policyLock.RUnlock()

	policiesRules := make([]market.AccessPolicyRuleSet, len(policies))
	for i, policy := range policies {
		index, exist := pr.getPolicyIndex(pr.policyList, policy)
		if !exist {
			return []market.AccessPolicyRuleSet{}, fmt.Errorf("unknown policy: %s", policy)
		}
		policiesRules[i] = pr.policyList[index].rules
	}
	return policiesRules, nil
}

func (pr *Repository) getPolicyIndex(policyList []policyMetadata, policy market.AccessPolicy) (int, bool) {
	for index, policyMeta := range policyList {
		if policyMeta.policy == policy {
			return index, true
		}
	}

	return 0, false
}

func (pr *Repository) fetchPolicyRules(policyMeta *policyMetadata) error {
	req, err := requests.NewGetRequest(policyMeta.policy.Source, "", nil)
	if err != nil {
		return errors.Wrap(err, "failed to create policy request")
	}
	req.Header.Add("If-None-Match", policyMeta.eTag)

	res, err := pr.client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed fetch policy rule %s", policyMeta.policy)
	}
	defer res.Body.Close()

	httptrace.TraceRequestResponse(req, res)

	if res.StatusCode == http.StatusNotModified {
		return nil
	}
	if err := requests.ParseResponseError(res); err != nil {
		return errors.Wrapf(err, "failed to fetch policy rule %s", policyMeta.policy)
	}

	policyMeta.rules = market.AccessPolicyRuleSet{}
	err = requests.ParseResponseJSON(res, &policyMeta.rules)
	if err != nil {
		return errors.Wrapf(err, "failed to parse policy rule %s", policyMeta.policy)
	}
	policyMeta.eTag = res.Header.Get("ETag")
	return nil
}

func (pr *Repository) fetchLoop() {
	for {
		select {
		case <-pr.fetchShutdown:
			return
		case <-time.After(pr.fetchInterval):
			pr.policyLock.Lock()

			policyListActive := make([]policyMetadata, len(pr.policyList))
			copy(policyListActive, pr.policyList)

			for index := range policyListActive {
				var err error
				if err = pr.fetchPolicyRules(&policyListActive[index]); err != nil {
					log.Warn().Err(err).Msg("synchronise fetch failed")
				}
			}
			pr.policyList = policyListActive

			pr.policyLock.Unlock()
		}
	}
}
