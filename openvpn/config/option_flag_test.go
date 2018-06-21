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

package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFlag_Factory(t *testing.T) {
	option := OptionFlag("enable-something")
	assert.NotNil(t, option)
}

func TestFlag_GetName(t *testing.T) {
	option := OptionFlag("enable-something")
	assert.Equal(t, "enable-something", option.getName())
}

func TestFlag_ToCli(t *testing.T) {
	option := OptionFlag("enable-something")

	optionValue, err := option.toCli()
	assert.NoError(t, err)
	assert.Equal(t, []string{"--enable-something"}, optionValue)
}

func TestFlag_ToFile(t *testing.T) {
	option := OptionFlag("enable-something")

	optionValue, err := option.toFile()
	assert.NoError(t, err)
	assert.Equal(t, "enable-something", optionValue)
}
