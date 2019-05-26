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

package reducer

import (
	"github.com/mysteriumnetwork/node/market"
)

// AndCondition defines sub condition given for And()
type AndCondition func(market.ServiceProposal) bool

// And matches proposal if all given conditions passes
func And(conditions ...AndCondition) func(market.ServiceProposal) bool {
	return func(proposal market.ServiceProposal) bool {
		var match bool
		for _, condition := range conditions {
			if match = condition(proposal); !match {
				break
			}
		}
		return match
	}
}
