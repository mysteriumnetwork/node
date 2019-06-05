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

// InString returns a matcher for checking if proposal's string field value exists in given array of values
func InString(field FieldSelector, valuesExpected ...string) func(market.ServiceProposal) bool {
	valuesExpectedTyped := make([]interface{}, len(valuesExpected))
	for i, value := range valuesExpected {
		valuesExpectedTyped[i] = value
	}

	return In(field, valuesExpectedTyped...)
}

// InInt returns a matcher for checking if proposal's integer field value exists in given array of values
func InInt(field FieldSelector, valuesExpected ...int) func(market.ServiceProposal) bool {
	valuesExpectedTyped := make([]interface{}, len(valuesExpected))
	for i, value := range valuesExpected {
		valuesExpectedTyped[i] = value
	}

	return In(field, valuesExpectedTyped...)
}

// In returns a matcher for checking if proposal's field value exists in given array of values
func In(field FieldSelector, valuesExpected ...interface{}) func(market.ServiceProposal) bool {
	return Field(field, func(value interface{}) bool {
		for _, valueExpected := range valuesExpected {
			if value == valueExpected {
				return true
			}
		}
		return false
	})
}
