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

type policySubscription struct {
	policy      market.AccessPolicy
	eTag        string
	subscribers []*Repository
}

// Oracle represents async policy fetcher from TrustOracle
type Oracle struct {
	client             *requests.HTTPClient
	fetchURL           string
	fetchInterval      time.Duration
	fetchLock          sync.RWMutex
	fetchSubscriptions []policySubscription

	fetchShutdown     chan struct{}
	fetchShutdownOnce sync.Once
}

// NewOracle create instance of policy fetcher
func NewOracle(client *requests.HTTPClient, policyURL string, interval time.Duration) *Oracle {
	return &Oracle{
		client:             client,
		fetchURL:           policyURL,
		fetchInterval:      interval,
		fetchSubscriptions: make([]policySubscription, 0),
		fetchShutdown:      make(chan struct{}),
	}
}

// Start begins fetching policies to subscribers
func (pr *Oracle) Start() {
	for {
		select {
		case <-pr.fetchShutdown:
			return
		case <-time.After(pr.fetchInterval):
			pr.fetchLock.Lock()

			subscriptionsActive := make([]policySubscription, len(pr.fetchSubscriptions))
			copy(subscriptionsActive, pr.fetchSubscriptions)

			for index := range subscriptionsActive {
				if err := pr.fetchPolicyRules(&subscriptionsActive[index]); err != nil {
					log.Warn().Err(err).Msg("synchronise fetch failed")
				}
			}
			pr.fetchSubscriptions = subscriptionsActive

			pr.fetchLock.Unlock()
		}
	}
}

// Stop ends fetching policies to subscribers
func (pr *Oracle) Stop() {
	pr.fetchShutdownOnce.Do(func() {
		close(pr.fetchShutdown)
	})
}

// Policy converts given value to valid policy rule
func (pr *Oracle) Policy(policyID string) market.AccessPolicy {
	policyURL := pr.fetchURL
	if !strings.HasSuffix(policyURL, "/") {
		policyURL += "/"
	}
	return market.AccessPolicy{
		ID:     policyID,
		Source: fmt.Sprintf("%v%v", policyURL, policyID),
	}
}

// Policies converts given values to list of valid policies
func (pr *Oracle) Policies(policyIDs []string) []market.AccessPolicy {
	policies := make([]market.AccessPolicy, len(policyIDs))
	for i, policyID := range policyIDs {
		policies[i] = pr.Policy(policyID)
	}
	return policies
}

// SubscribePolicies adds given policies to repository and syncs changes of it's items from TrustOracle
func (pr *Oracle) SubscribePolicies(policies []market.AccessPolicy, repository *Repository) error {
	pr.fetchLock.Lock()
	defer pr.fetchLock.Unlock()

	subscriptionsNew := make([]policySubscription, len(pr.fetchSubscriptions))
	copy(subscriptionsNew, pr.fetchSubscriptions)

	for _, policy := range policies {
		index := len(subscriptionsNew)
		subscriptionsNew = append(subscriptionsNew, policySubscription{
			policy:      policy,
			subscribers: []*Repository{repository},
		})

		if err := pr.fetchPolicyRules(&subscriptionsNew[index]); err != nil {
			return errors.Wrap(err, "initial fetch failed")
		}
	}

	pr.fetchSubscriptions = subscriptionsNew
	return nil
}

func (pr *Oracle) fetchPolicyRules(subscription *policySubscription) error {
	req, err := requests.NewGetRequest(subscription.policy.Source, "", nil)
	if err != nil {
		return errors.Wrap(err, "failed to create policy request")
	}
	req.Header.Add("If-None-Match", subscription.eTag)

	res, err := pr.client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed fetch policy rule %s", subscription.policy)
	}
	defer res.Body.Close()

	httptrace.TraceRequestResponse(req, res)

	if res.StatusCode == http.StatusNotModified {
		return nil
	}
	if err := requests.ParseResponseError(res); err != nil {
		return errors.Wrapf(err, "failed to fetch policy rule %s", subscription.policy)
	}

	var rules = market.AccessPolicyRuleSet{}
	err = requests.ParseResponseJSON(res, &rules)
	if err != nil {
		return errors.Wrapf(err, "failed to parse policy rule %s", subscription.policy)
	}
	subscription.eTag = res.Header.Get("ETag")

	for _, subscriber := range subscription.subscribers {
		subscriber.SetPolicyRules(subscription.policy, rules)
	}

	return nil
}
