/*
 * Copyright (C) 2023 The "MysteriumNetwork/node" Authors.
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

package requested

import (
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/rs/zerolog/log"
)

// Provider defines the requested provider structure.
type Provider struct {
	client   *requests.HTTPClient
	fetchURL string
}

// NewRequestedProvider creates a policy provider
func NewRequestedProvider(client *requests.HTTPClient, policyURL string) *Provider {
	return &Provider{
		client:   client,
		fetchURL: policyURL,
	}
}

// IsIdentityAllowed returns if provided identity exists in any access policy.
func (o *Provider) IsIdentityAllowed(identity identity.Identity) bool {
	req, err := requests.NewGetRequest(o.fetchURL, "", nil)
	if err != nil {
		log.Warn().Err(err).Msg("failed to create policy request")
		return false
	}

	queryValues := req.URL.Query()
	queryValues.Add("identity-value", identity.Address)
	req.URL.RawQuery = queryValues.Encode()

	resp, err := o.client.Do(req)
	if err != nil {
		log.Warn().Err(err).Msg("failed to make policy request")
		return false
	}

	return resp.StatusCode >= 200 && resp.StatusCode <= 299
}

// HasDNSRules returns if dns rules exist. Currently unsupported with this implemenetation
func (o *Provider) HasDNSRules() bool {
	return false
}

// IsHostAllowed returns if provided host is allowed. Currently unsupported with this implemenetation
func (o *Provider) IsHostAllowed(host string) bool {
	return false
}
