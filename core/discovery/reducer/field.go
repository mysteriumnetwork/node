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

// FieldSelector callback to select field in proposal
type FieldSelector func(proposal market.ServiceProposal) interface{}

// FieldCondition returns flag if field value matches against it's rules
type FieldCondition func(value interface{}) bool

// Field matches proposal if given FieldCondition passes
func Field(field FieldSelector, reducer FieldCondition) func(market.ServiceProposal) bool {
	return func(proposal market.ServiceProposal) bool {
		return reducer(field(proposal))
	}
}

// Empty matches proposal if given field is empty
func Empty(field FieldSelector) func(market.ServiceProposal) bool {
	return Field(field, func(value interface{}) bool {
		switch valueTyped := value.(type) {
		case string:
			return valueTyped == ""
		case int:
			return valueTyped == 0
		case byte:
			return valueTyped == byte(0)
		case bool:
			return valueTyped == false
		case []string:
		case []int:
		case []byte:
		case []bool:
			return len(valueTyped) == 0
		}
		return false
	})
}
