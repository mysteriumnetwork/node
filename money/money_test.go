/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewMoney(t *testing.T) {
	assert.Equal(
		t,
		uint64(0.150*100000000),
		NewMoney(0.150, CurrencyMyst).Amount,
	)

	assert.Equal(
		t,
		uint64(0*100000000),
		NewMoney(0, CurrencyMyst).Amount,
	)

	assert.Equal(
		t,
		uint64(10*100000000),
		NewMoney(10, CurrencyMyst).Amount,
	)

	assert.NotEqual(
		t,
		uint64(1),
		NewMoney(1, CurrencyMyst).Amount,
	)
}
