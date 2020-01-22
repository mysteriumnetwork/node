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
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
)

// ValidateAllowedIdentity checks if given identity is allowed by given policies
func ValidateAllowedIdentity(repository *PolicyRepository, policies *[]market.AccessPolicy) func(identity.Identity) error {
	return func(peerID identity.Identity) error {
		if policies == nil {
			return nil
		}

		policiesRules, err := repository.RulesForPolicies(*policies)
		if err != nil {
			return err
		}

		for _, policyRules := range policiesRules {
			for _, rule := range policyRules.Allow {
				if rule.Type == market.AccessPolicyTypeIdentity && rule.Value == peerID.Address {
					return nil
				}
			}
		}

		return nil
	}
}
