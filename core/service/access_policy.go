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

package service

import (
	"time"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/pkg/errors"
)

func fetchAllowedIDs(ap *[]market.AccessPolicy) (allowedIDs []identity.Identity, err error) {
	if ap == nil {
		return nil, nil
	}

	client := requests.NewHTTPClient(time.Minute)

	var ruleSet market.AccessPolicyRuleSet
	for _, p := range *ap {
		req, err := requests.NewGetRequest(p.Source, "", nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create request")
		}

		err = client.DoRequestAndParseResponse(req, &ruleSet)
		if err != nil {
			return nil, errors.Wrap(err, "failed to execute request")
		}

		for _, rule := range ruleSet.Allow {
			if rule.Type == "identity" {
				allowedIDs = append(allowedIDs, identity.FromAddress(rule.Value))
			}
		}
	}

	return allowedIDs, nil
}
