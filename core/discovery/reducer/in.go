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

// In matches proposal if field value exists in array
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

// InString matches proposal if string value exists in array
func InString(field FieldSelector, valuesExpected ...string) func(market.ServiceProposal) bool {
	valuesExpectedTyped := make([]interface{}, len(valuesExpected))
	for i, value := range valuesExpected {
		valuesExpectedTyped[i] = value
	}

	return In(field, valuesExpectedTyped...)
}
