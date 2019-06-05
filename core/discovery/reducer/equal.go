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

// EqualInt returns a matcher for checking if proposal's integer field value equal to given value
func EqualInt(field FieldSelector, valueExpected int) func(market.ServiceProposal) bool {
	return Equal(field, valueExpected)
}

// EqualString returns a matcher for checking if proposal's string field value equal to given value
func EqualString(field FieldSelector, valueExpected string) func(market.ServiceProposal) bool {
	return Equal(field, valueExpected)
}

// Equal returns a matcher for checking if proposal's field value equal to given value
func Equal(field FieldSelector, valueExpected interface{}) func(market.ServiceProposal) bool {
	return Field(field, func(value interface{}) bool {
		return value == valueExpected
	})
}
