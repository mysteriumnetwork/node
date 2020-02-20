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

package datasize

import (
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBits(t *testing.T) {
	table := []struct {
		value         BitSize
		valueExpected uint64
	}{
		{B, 8},
		{10 * B, 80},
		{0.5 * B, 4},
		{0.2 * B, 1},
		{0.1 * B, 0},
	}

	for _, tt := range table {
		assert.Equal(t, tt.valueExpected, tt.value.Bits())
	}
}

func TestBytes(t *testing.T) {
	table := []struct {
		value         BitSize
		valueExpected uint64
	}{
		{KiB, 1024},
		{10 * KiB, 10240},
		{0.5 * KiB, 512},
		{0.001 * KiB, 1},
	}

	for _, tt := range table {
		assert.Equal(t, tt.valueExpected, tt.value.Bytes())
	}
}

func Test_String(t *testing.T) {
	tests := []struct {
		input BitSize
		want  string
	}{
		{0, "0 b"},
		{Bit, "1 b"},
		{B, "1 B"},
		{KiB, "1.0 KiB"},
		{MiB, "1.0 MiB"},
		{GiB, "1.0 GiB"},
		{TiB, "1.0 TiB"},
		{PiB, "1.0 PiB"},
		{EiB, "1.0 EiB"},
		{400 * TiB, "400.0 TiB"},
		{2048 * MiB, "2.0 GiB"},
		{B + KiB, "1.0 KiB"},
		{MiB + 20*KiB, "1.0 MiB"},
		{100*MiB + KiB, "100.0 MiB"},
		{50 * B, "50 B"},
		{1024 * B, "1.0 KiB"},
		{1500 * B, "1.5 KiB"},
		{1024 * 1024 * B, "1.0 MiB"},
		{1024 * 1024 * 1024 * B, "1.0 GiB"},
		{BitSize(math.MaxUint64 + 1), "2.0 EiB"},
	}
	for idx, tt := range tests {
		t.Run(strconv.Itoa(idx), func(t *testing.T) {
			result := tt.input.String()
			assert.Equal(t, tt.want, result)
		})
	}
}
