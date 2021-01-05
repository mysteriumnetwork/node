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
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var myst = common.HexToAddress("0x0")
var dai = common.HexToAddress("0x1")
var weth = common.HexToAddress("0x2")

func Test_Exchange_MystToDai(t *testing.T) {
	c := &mockUniswapClient{
		exchangeForPath: oneDai,
	}
	e := NewExchange(myst, dai, weth, c)

	r, err := e.MystToDai()
	assert.NoError(t, err)

	assert.Equal(t, 1.0, r)

	mockErr := errors.New("boom")
	c.errForPath = mockErr

	_, err = e.MystToDai()
	assert.Equal(t, mockErr, err)
}

func Test_Exchange_DaiToMyst(t *testing.T) {
	c := &mockUniswapClient{
		exchangeForPath: oneMyst,
	}
	e := NewExchange(myst, dai, weth, c)

	r, err := e.DaiToMyst()
	assert.NoError(t, err)

	assert.Equal(t, 1.0, r)

	mockErr := errors.New("boom")
	c.errForPath = mockErr

	_, err = e.DaiToMyst()
	assert.Equal(t, mockErr, err)
}

type mockUniswapClient struct {
	exchangeForPath *big.Int
	errForPath      error
}

func (mu *mockUniswapClient) GetExchangeAmountForPath(amount *big.Int, tokens ...common.Address) (*big.Int, error) {
	return mu.exchangeForPath, mu.errForPath
}
