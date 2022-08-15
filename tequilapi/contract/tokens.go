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
	"math/big"

	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/shopspring/decimal"
)

// TokensZero zero amount
var TokensZero = Tokens{Wei: "0", Ether: "0", Human: "0"}

// HumanPrecision default human ether amount precision
const HumanPrecision = 6

// Tokens a common response for ethereum blockchain monetary amount
type Tokens struct {
	Wei   string `json:"wei"`
	Ether string `json:"ether"`
	Human string `json:"human"`
}

// NewTokens convenience constructor for Tokens
func NewTokens(amount *big.Int) Tokens {
	if amount == nil {
		return TokensZero
	}

	ethers := crypto.BigMystToDecimal(amount)

	return Tokens{
		Wei:   amount.String(),
		Ether: ethers.String(),
		Human: ethers.Truncate(HumanPrecision).String(),
	}
}

// NewTokensFromDecimal convenience constructor for Tokens
func NewTokensFromDecimal(amount decimal.Decimal) Tokens {
	return Tokens{
		Wei:   crypto.DecimalToBigMyst(amount).String(),
		Ether: amount.String(),
		Human: amount.Truncate(HumanPrecision).String(),
	}
}

func (t Tokens) String() string {
	return t.Human
}
