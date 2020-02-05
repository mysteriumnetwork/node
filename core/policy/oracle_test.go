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
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/stretchr/testify/assert"
)

func Test_Oracle_Policy(t *testing.T) {
	repo := &Oracle{fetchURL: "http://policy.localhost"}
	assert.Equal(
		t,
		market.AccessPolicy{ID: "1", Source: "http://policy.localhost/1"},
		repo.Policy("1"),
	)

	repo = &Oracle{fetchURL: "http://policy.localhost/"}
	assert.Equal(
		t,
		market.AccessPolicy{ID: "2", Source: "http://policy.localhost/2"},
		repo.Policy("2"),
	)
}

func Test_Oracle_Policies(t *testing.T) {
	repo := &Oracle{fetchURL: "http://policy.localhost"}
	assert.Equal(
		t,
		[]market.AccessPolicy{
			{ID: "1", Source: "http://policy.localhost/1"},
		},
		repo.Policies([]string{"1"}),
	)

	repo = &Oracle{fetchURL: "http://policy.localhost/"}
	assert.Equal(
		t,
		[]market.AccessPolicy{
			{ID: "2", Source: "http://policy.localhost/2"},
			{ID: "3", Source: "http://policy.localhost/3"},
		},
		repo.Policies([]string{"2", "3"}),
	)
}

func Test_Oracle_SubscribePolicies_WhenEndpointFails(t *testing.T) {
	repo := NewRepository()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	oracle := createEmptyOracle(server.URL)
	err := oracle.SubscribePolicies(
		[]market.AccessPolicy{oracle.Policy("1"), oracle.Policy("3")},
		repo,
	)
	assert.EqualError(
		t,
		err,
		fmt.Sprintf("initial fetch failed: failed to fetch policy rule {1 %s/1}: server response invalid: 500 Internal Server Error (%s/1)", server.URL, server.URL),
	)
	assert.Equal(t, []market.AccessPolicy{}, repo.Policies())
}

func Test_Oracle_SubscribePolicies_Race(t *testing.T) {
	repo := NewRepository()
	server := mockPolicyServer()
	defer server.Close()
	oracle := createEmptyOracle(server.URL)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		err := oracle.SubscribePolicies(
			[]market.AccessPolicy{oracle.Policy("1"), oracle.Policy("3")},
			repo,
		)
		assert.NoError(t, err)
	}()
	go func() {
		defer wg.Done()
		err := oracle.SubscribePolicies([]market.AccessPolicy{oracle.Policy("2")}, repo)
		assert.NoError(t, err)
	}()
	wg.Wait()

	policiesRules, err := repo.RulesForPolicies([]market.AccessPolicy{
		oracle.Policy("1"),
		oracle.Policy("2"),
		oracle.Policy("3"),
	})
	assert.NoError(t, err)
	assert.Len(t, policiesRules, 3)
}

func Test_Oracle_SubscribePolicies_WhenEndpointSucceeds(t *testing.T) {
	repo := NewRepository()
	server := mockPolicyServer()
	defer server.Close()

	oracle := createEmptyOracle(server.URL)
	err := oracle.SubscribePolicies(
		[]market.AccessPolicy{oracle.Policy("1"), oracle.Policy("3")},
		repo,
	)
	assert.NoError(t, err)
	policiesRules, err := repo.RulesForPolicies([]market.AccessPolicy{
		oracle.Policy("1"),
		oracle.Policy("3"),
	})
	assert.NoError(t, err)
	assert.Equal(
		t,
		[]market.AccessPolicyRuleSet{policyOneRulesUpdated, policyThreeRulesUpdated},
		policiesRules,
	)

	oracle = createFilledOracle(server.URL, time.Minute, repo)
	err = oracle.SubscribePolicies(
		[]market.AccessPolicy{oracle.Policy("1"), oracle.Policy("3")},
		repo,
	)
	assert.NoError(t, err)

	policiesRules, err = repo.RulesForPolicies([]market.AccessPolicy{
		oracle.Policy("1"),
		oracle.Policy("3"),
	})
	assert.NoError(t, err)
	assert.Equal(
		t,
		[]market.AccessPolicyRuleSet{policyOneRulesUpdated, policyThreeRulesUpdated},
		policiesRules,
	)
}

func Test_Oracle_StartSyncsPolicies(t *testing.T) {
	repo := NewRepository()
	server := mockPolicyServer()
	defer server.Close()

	oracle := createFilledOracle(server.URL, 1*time.Millisecond, repo)
	oracle.Start()
	defer oracle.Stop()

	var policiesRules []market.AccessPolicyRuleSet
	assert.Eventually(t, func() bool {
		var err error
		policiesRules, err = repo.RulesForPolicies([]market.AccessPolicy{
			oracle.Policy("1"),
			oracle.Policy("2"),
		})
		return err == nil && len(policiesRules) == 2
	}, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, []market.AccessPolicyRuleSet{policyOneRulesUpdated, policyTwoRulesUpdated}, policiesRules)
}

func Test_PolicyRepository_StartMultipleTimes(t *testing.T) {
	oracle := NewOracle(requests.NewHTTPClient("0.0.0.0", time.Second), "http://policy.localhost", time.Minute)
	oracle.Start()
	oracle.Stop()

	oracle.Start()
	oracle.Stop()
}

func createEmptyOracle(mockServerURL string) *Oracle {
	return NewOracle(
		requests.NewHTTPClient("0.0.0.0", 100*time.Millisecond),
		mockServerURL+"/",
		time.Minute,
	)
}

func createFilledOracle(mockServerURL string, interval time.Duration, repo *Repository) *Oracle {
	oracle := NewOracle(
		requests.NewHTTPClient("0.0.0.0", time.Second),
		mockServerURL+"/",
		interval,
	)
	oracle.SubscribePolicies(
		[]market.AccessPolicy{oracle.Policy("1"), oracle.Policy("2")},
		repo,
	)
	return oracle
}

func mockPolicyServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "1",
				"title": "One (updated)",
				"description": "",
				"allow": [
					{"type": "identity", "value": "0x1"}
				]
			}`))
		} else if r.URL.Path == "/2" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "2",
				"title": "Two (updated)",
				"description": "",
				"allow": [
					{"type": "dns_hostname", "value": "ipinfo.io"}
				]
			}`))
		} else if r.URL.Path == "/3" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "3",
				"title": "Three (updated)",
				"description": "",
				"allow": [
					{"type": "dns_zone", "value": "ipinfo.io"}
				]
			}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
