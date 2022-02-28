/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package connection

import (
	"fmt"
	"time"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
)

type proposalRepository interface {
	Proposals(filter *proposal.Filter) ([]proposal.PricedServiceProposal, error)
}

// FilteredProposals create an function to keep getting proposals from the discovery based on the provided filters.
func FilteredProposals(f *proposal.Filter, sortBy string, repo proposalRepository) func() (*proposal.PricedServiceProposal, error) {
	usedProposals := make(map[string]time.Time)

	return func() (*proposal.PricedServiceProposal, error) {
		proposals, err := repo.Proposals(f)
		if err != nil {
			return nil, err
		}

		proposals, err = proposal.Sort(proposals, sortBy)
		if err != nil {
			return nil, fmt.Errorf("failed to sort proposals: %w", err)
		}

		for _, p := range proposals { // Trying to find providers that we didn't try to connect during 5 minutes.
			if t, ok := usedProposals[p.ProviderID]; !ok || time.Since(t) > 5*time.Minute {
				usedProposals[p.ProviderID] = time.Now()
				return &p, nil
			}
		}

		for _, p := range proposals { // If we failed to find new provider, trying the old ones.
			usedProposals[p.ProviderID] = time.Now()
			return &p, nil
		}

		return nil, fmt.Errorf("no providers available for the filter")
	}
}
