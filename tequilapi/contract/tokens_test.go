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

package contract

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokens(t *testing.T) {
	for _, data := range []struct {
		amount        *big.Int
		expectedWei   string
		expectedEther string
		expectedHuman string
	}{
		{
			amount:        big.NewInt(6_123_456_789_123_456_789),
			expectedWei:   "6123456789123456789",
			expectedEther: "6.123456789123456789",
			expectedHuman: "6.123456", // hence no rounding
		},
		{
			amount:        big.NewInt(0),
			expectedWei:   "0",
			expectedEther: "0",
			expectedHuman: "0",
		},
		{
			amount:        big.NewInt(1),
			expectedWei:   "1",
			expectedEther: "0.000000000000000001",
			expectedHuman: "0",
		},
		{
			amount:        big.NewInt(-1),
			expectedWei:   "-1",
			expectedEther: "-0.000000000000000001",
			expectedHuman: "0",
		},
		{
			amount:        big.NewInt(-6_123_456_789_123_456_789),
			expectedWei:   "-6123456789123456789",
			expectedEther: "-6.123456789123456789",
			expectedHuman: "-6.123456", // hence no rounding
		},
		{
			amount:        nil,
			expectedWei:   "0",
			expectedEther: "0",
			expectedHuman: "0",
		},
	} {
		t.Run(fmt.Sprintf("%+v", data), func(t *testing.T) {
			tokens := NewTokens(data.amount)
			assert.Equal(t, tokens.Wei, data.expectedWei)
			assert.Equal(t, tokens.Ether, data.expectedEther)
			assert.Equal(t, tokens.Human, data.expectedHuman)
		})
	}
}
