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

package money

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/payments/crypto"
)

type uniswapClient interface {
	GetExchangeAmountForPath(amount *big.Int, tokens ...common.Address) (*big.Int, error)
}

// Exchange allows to check the prices of various tokens in relation to one another.
type Exchange struct {
	mystAddress common.Address
	daiAddress  common.Address
	wethAddress common.Address
	c           uniswapClient
}

// NewExchange creates a new instance of exchange.
func NewExchange(mystAddress, daiAddress, wethAddress common.Address, c uniswapClient) *Exchange {
	return &Exchange{
		mystAddress: mystAddress,
		daiAddress:  daiAddress,
		wethAddress: wethAddress,
		c:           c,
	}
}

// both myst and dai have 18 zeroes.
var oneMyst = big.NewInt(0).SetUint64(crypto.Myst)
var oneDai = big.NewInt(0).SetUint64(crypto.Myst)

// MystToDai returns the amount of dai you'd get for a single myst via uniswap.
func (e *Exchange) MystToDai() (float64, error) {
	resultingDai, err := e.c.GetExchangeAmountForPath(oneMyst, e.mystAddress, e.wethAddress, e.daiAddress)
	if err != nil {
		return 0, err
	}
	return crypto.BigMystToFloat(resultingDai), nil
}

// DaiToMyst returns the amount of myst you'd get for a single dai via uniswap.
func (e *Exchange) DaiToMyst() (float64, error) {
	resultingMyst, err := e.c.GetExchangeAmountForPath(oneDai, e.daiAddress, e.wethAddress, e.mystAddress)
	if err != nil {
		return 0, err
	}
	return crypto.BigMystToFloat(resultingMyst), nil
}
