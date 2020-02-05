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

var (
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

	policyThreeRulesUpdated = market.AccessPolicyRuleSet{
		ID:    "3",
		Title: "Three (updated)",
		Allow: []market.AccessRule{
			{Type: market.AccessPolicyTypeDNSZone, Value: "ipinfo.io"},
		},
	}
)

func Test_PolicyRepository_Policy(t *testing.T) {
	repo := &Repository{policyURL: "http://policy.localhost"}
	assert.Equal(
		t,
		market.AccessPolicy{ID: "1", Source: "http://policy.localhost/1"},
		repo.Policy("1"),
	)

	repo = &Repository{policyURL: "http://policy.localhost/"}
	assert.Equal(
		t,
		market.AccessPolicy{ID: "2", Source: "http://policy.localhost/2"},
		repo.Policy("2"),
	)
}

func Test_PolicyRepository_Policies(t *testing.T) {
	repo := &Repository{policyURL: "http://policy.localhost"}
	assert.Equal(
		t,
		&[]market.AccessPolicy{
			{ID: "1", Source: "http://policy.localhost/1"},
		},
		repo.Policies([]string{"1"}),
	)

	repo = &Repository{policyURL: "http://policy.localhost/"}
	assert.Equal(
		t,
		&[]market.AccessPolicy{
			{ID: "2", Source: "http://policy.localhost/2"},
			{ID: "3", Source: "http://policy.localhost/3"},
		},
		repo.Policies([]string{"2", "3"}),
	)
}

func Test_PolicyRepository_RulesForPolicy(t *testing.T) {
	repo := createEmptyRepo("http://policy.localhost")
	policyRules, err := repo.RulesForPolicy(repo.Policy("1"))
	assert.EqualError(t, err, "unknown policy: {1 http://policy.localhost/1}")
	assert.Equal(t, market.AccessPolicyRuleSet{}, policyRules)

	repo = createFullRepo("http://policy.localhost", time.Minute)
	policyRules, err = repo.RulesForPolicy(repo.Policy("1"))
	assert.NoError(t, err)
	assert.Equal(t, policyOneRules, policyRules)
}

func Test_PolicyRepository_RulesForPolicies(t *testing.T) {
	repo := createEmptyRepo("http://policy.localhost")
	policiesRules, err := repo.RulesForPolicies([]market.AccessPolicy{
		repo.Policy("1"),
		repo.Policy("3"),
	})
	assert.EqualError(t, err, "unknown policy: {1 http://policy.localhost/1}")
	assert.Equal(t, []market.AccessPolicyRuleSet{}, policiesRules)

	repo = createFullRepo("http://policy.localhost", time.Minute)
	policiesRules, err = repo.RulesForPolicies([]market.AccessPolicy{
		repo.Policy("1"),
		repo.Policy("2"),
	})
	assert.NoError(t, err)
	assert.Equal(t, []market.AccessPolicyRuleSet{policyOneRules, policyTwoRules}, policiesRules)
}

func Test_PolicyRepository_AddPolicies_WhenEndpointFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	repo := createEmptyRepo(server.URL)
	err := repo.AddPolicies([]market.AccessPolicy{
		repo.Policy("1"),
		repo.Policy("3"),
	})
	assert.EqualError(
		t,
		err,
		fmt.Sprintf("initial fetch failed: failed to fetch policy rule {1 %s/1}: server response invalid: 500 Internal Server Error (%s/1)", server.URL, server.URL),
	)
	assert.Equal(t, []policyMetadata{}, repo.policyList)
}

func Test_PolicyRepository_AddPolicies_Race(t *testing.T) {
	server := mockPolicyServer()
	defer server.Close()
	repo := createEmptyRepo(server.URL)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		err := repo.AddPolicies([]market.AccessPolicy{
			repo.Policy("1"),
			repo.Policy("3"),
		})
		assert.NoError(t, err)
	}()
	go func() {
		defer wg.Done()
		err := repo.AddPolicies([]market.AccessPolicy{
			repo.Policy("2"),
		})
		assert.NoError(t, err)
	}()
	wg.Wait()

	policiesRules, err := repo.RulesForPolicies([]market.AccessPolicy{
		repo.Policy("1"),
		repo.Policy("2"),
		repo.Policy("3"),
	})
	assert.NoError(t, err)
	assert.Len(t, policiesRules, 3)
}

func Test_PolicyRepository_AddPolicies_WhenEndpointSucceeds(t *testing.T) {
	server := mockPolicyServer()
	defer server.Close()

	repo := createEmptyRepo(server.URL)
	err := repo.AddPolicies([]market.AccessPolicy{
		repo.Policy("1"),
		repo.Policy("3"),
	})
	assert.NoError(t, err)
	assert.Equal(
		t,
		[]policyMetadata{
			{policy: repo.Policy("1"), rules: policyOneRulesUpdated},
			{policy: repo.Policy("3"), rules: policyThreeRulesUpdated},
		},
		repo.policyList,
	)

	repo = createFullRepo(server.URL, time.Minute)
	err = repo.AddPolicies([]market.AccessPolicy{
		repo.Policy("1"),
		repo.Policy("3"),
	})
	assert.NoError(t, err)
	assert.Equal(
		t,
		[]policyMetadata{
			{policy: repo.Policy("1"), rules: policyOneRulesUpdated},
			{policy: repo.Policy("2"), rules: policyTwoRules},
			{policy: repo.Policy("3"), rules: policyThreeRulesUpdated},
		},
		repo.policyList,
	)
}

func Test_PolicyRepository_StartSyncsPolicies(t *testing.T) {
	server := mockPolicyServer()
	defer server.Close()

	repo := createFullRepo(server.URL, 1*time.Millisecond)
	repo.Start()
	defer repo.Stop()

	time.Sleep(10 * time.Millisecond)
	policiesRules, err := repo.RulesForPolicies([]market.AccessPolicy{
		repo.Policy("1"),
		repo.Policy("2"),
	})
	assert.NoError(t, err)
	assert.Equal(t, []market.AccessPolicyRuleSet{policyOneRulesUpdated, policyTwoRulesUpdated}, policiesRules)
}

func Test_PolicyRepository_StartMultipleTimes(t *testing.T) {
	repo := NewRepository(requests.NewHTTPClient("0.0.0.0", time.Second), "http://policy.localhost", time.Minute)
	repo.Start()
	repo.Stop()

	repo.Start()
	repo.Stop()
}

func createEmptyRepo(mockServerURL string) *Repository {
	return NewRepository(
		requests.NewHTTPClient("0.0.0.0", 100*time.Millisecond),
		mockServerURL+"/",
		time.Minute,
	)
}

func createFullRepo(mockServerURL string, interval time.Duration) *Repository {
	repo := NewRepository(
		requests.NewHTTPClient("0.0.0.0", time.Second),
		mockServerURL+"/",
		interval,
	)
	repo.policyList = append(
		repo.policyList,
		policyMetadata{
			policy: repo.Policy("1"),
			rules: market.AccessPolicyRuleSet{
				ID:    "1",
				Title: "One",
				Allow: []market.AccessRule{
					{Type: market.AccessPolicyTypeIdentity, Value: "0x1"},
				},
			},
		},
		policyMetadata{
			policy: repo.Policy("2"),
			rules: market.AccessPolicyRuleSet{
				ID:    "2",
				Title: "Two",
				Allow: []market.AccessRule{
					{Type: market.AccessPolicyTypeDNSHostname, Value: "ipinfo.io"},
				},
			},
		},
	)
	return repo
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
