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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstants(t *testing.T) {
	table := []struct {
		value        BitSize
		expectedBits uint64
	}{
		{b, 1},
		{Bit, 1},

		{Byte, 8},
		{B, 8},

		{Kibibyte, 8 * 1024},
		{KiB, 8 * 1024},

		{Mebibyte, 8 * 1024 * 1024},
		{MiB, 8 * 1024 * 1024},

		{Gibibyte, 8 * 1024 * 1024 * 1024},
		{GiB, 8 * 1024 * 1024 * 1024},

		{Tebibyte, 8 * 1024 * 1024 * 1024 * 1024},
		{TiB, 8 * 1024 * 1024 * 1024 * 1024},

		{Pebibyte, 8 * 1024 * 1024 * 1024 * 1024 * 1024},
		{PiB, 8 * 1024 * 1024 * 1024 * 1024 * 1024},

		{Exbibyte, 8 * 1024 * 1024 * 1024 * 1024 * 1024 * 1024},
		{EiB, 8 * 1024 * 1024 * 1024 * 1024 * 1024 * 1024},
	}

	for _, tt := range table {
		assert.Equal(t, tt.expectedBits, uint64(tt.value))
	}
}

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
		valueExpected float64
	}{
		{KiB, 1024},
		{10 * KiB, 10240},
		{0.5 * KiB, 512},
		{0.001 * KiB, 1.024},
	}

	for _, tt := range table {
		assert.Equal(t, tt.valueExpected, tt.value.Bytes())
	}
}

func TestKibibytes(t *testing.T) {
	table := []struct {
		value         BitSize
		valueExpected float64
	}{
		{MiB, 1024},
		{10 * MiB, 10240},
		{0.5 * MiB, 512},
		{0.001 * MiB, 1.024},
	}

	for _, tt := range table {
		assert.Equal(t, tt.valueExpected, tt.value.Kibibytes())
	}
}

func TestMebibytes(t *testing.T) {
	table := []struct {
		value         BitSize
		valueExpected float64
	}{
		{GiB, 1024},
		{10 * GiB, 10240},
		{0.5 * GiB, 512},
		{0.001 * GiB, 1.024},
	}

	for _, tt := range table {
		assert.Equal(t, tt.valueExpected, tt.value.Mebibytes())
	}
}

func TestGibibytes(t *testing.T) {
	table := []struct {
		value         BitSize
		valueExpected float64
	}{
		{TiB, 1024},
		{10 * TiB, 10240},
		{0.5 * TiB, 512},
		{0.001 * TiB, 1.024},
	}

	for _, tt := range table {
		assert.Equal(t, tt.valueExpected, tt.value.Gibibytes())
	}
}

func TestTebibytes(t *testing.T) {
	table := []struct {
		value         BitSize
		valueExpected float64
	}{
		{PiB, 1024},
		{10 * PiB, 10240},
		{0.5 * PiB, 512},
		{0.001 * PiB, 1.024},
	}

	for _, tt := range table {
		assert.Equal(t, tt.valueExpected, tt.value.Tebibytes())
	}
}

func TestPebibytes(t *testing.T) {
	table := []struct {
		value         BitSize
		valueExpected float64
	}{
		{EiB, 1024},
		{10 * EiB, 10240},
		{0.5 * EiB, 512},
		{0.001 * EiB, 1.024},
	}

	for _, tt := range table {
		assert.Equal(t, tt.valueExpected, tt.value.Pebibytes())
	}
}

func TestExbibytes(t *testing.T) {
	table := []struct {
		value         BitSize
		valueExpected float64
	}{
		{EiB, 1},
		{10 * EiB, 10},
		{0.5 * EiB, 0.5},
		{0.001 * EiB, 0.001},
	}

	for _, tt := range table {
		assert.Equal(t, tt.valueExpected, tt.value.Exbibytes())
	}
}

func TestMarshalText(t *testing.T) {
	table := []struct {
		value       BitSize
		valueString string
	}{
		{0, "0b"},
		{Bit, "1b"},
		{B, "1B"},
		{KiB, "1KiB"},
		{MiB, "1MiB"},
		{GiB, "1GiB"},
		{TiB, "1TiB"},
		{PiB, "1PiB"},
		{EiB, "1EiB"},
		{400 * TiB, "400TiB"},
		{2048 * MiB, "2GiB"},
		{B + KiB, "1025B"},
		{MiB + 20*KiB, "1044KiB"},
		{100*MiB + KiB, "102401KiB"},
	}

	for _, tt := range table {
		assert.Equal(t, tt.valueString, tt.value.String())
	}
}
