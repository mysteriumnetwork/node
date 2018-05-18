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
	"github.com/stretchr/testify/assert"
	"testing"
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

		{Kilobyte, 8 * 1024},
		{KB, 8 * 1024},

		{Megabyte, 8 * 1024 * 1024},
		{MB, 8 * 1024 * 1024},

		{Gigabyte, 8 * 1024 * 1024 * 1024},
		{GB, 8 * 1024 * 1024 * 1024},

		{Terabyte, 8 * 1024 * 1024 * 1024 * 1024},
		{TB, 8 * 1024 * 1024 * 1024 * 1024},

		{Petabyte, 8 * 1024 * 1024 * 1024 * 1024 * 1024},
		{PB, 8 * 1024 * 1024 * 1024 * 1024 * 1024},

		{Exabyte, 8 * 1024 * 1024 * 1024 * 1024 * 1024 * 1024},
		{EB, 8 * 1024 * 1024 * 1024 * 1024 * 1024 * 1024},
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
		{KB, 1024},
		{10 * KB, 10240},
		{0.5 * KB, 512},
		{0.001 * KB, 1.024},
	}

	for _, tt := range table {
		assert.Equal(t, tt.valueExpected, tt.value.Bytes())
	}
}

func TestKilobytes(t *testing.T) {
	table := []struct {
		value         BitSize
		valueExpected float64
	}{
		{MB, 1024},
		{10 * MB, 10240},
		{0.5 * MB, 512},
		{0.001 * MB, 1.024},
	}

	for _, tt := range table {
		assert.Equal(t, tt.valueExpected, tt.value.Kilobytes())
	}
}

func TestMegabytes(t *testing.T) {
	table := []struct {
		value         BitSize
		valueExpected float64
	}{
		{GB, 1024},
		{10 * GB, 10240},
		{0.5 * GB, 512},
		{0.001 * GB, 1.024},
	}

	for _, tt := range table {
		assert.Equal(t, tt.valueExpected, tt.value.Megabytes())
	}
}

func TestGigabytes(t *testing.T) {
	table := []struct {
		value         BitSize
		valueExpected float64
	}{
		{TB, 1024},
		{10 * TB, 10240},
		{0.5 * TB, 512},
		{0.001 * TB, 1.024},
	}

	for _, tt := range table {
		assert.Equal(t, tt.valueExpected, tt.value.Gigabytes())
	}
}

func TestTerabytes(t *testing.T) {
	table := []struct {
		value         BitSize
		valueExpected float64
	}{
		{PB, 1024},
		{10 * PB, 10240},
		{0.5 * PB, 512},
		{0.001 * PB, 1.024},
	}

	for _, tt := range table {
		assert.Equal(t, tt.valueExpected, tt.value.Terabytes())
	}
}

func TestPetabytes(t *testing.T) {
	table := []struct {
		value         BitSize
		valueExpected float64
	}{
		{EB, 1024},
		{10 * EB, 10240},
		{0.5 * EB, 512},
		{0.001 * EB, 1.024},
	}

	for _, tt := range table {
		assert.Equal(t, tt.valueExpected, tt.value.Petabytes())
	}
}

func TestExabytes(t *testing.T) {
	table := []struct {
		value         BitSize
		valueExpected float64
	}{
		{EB, 1},
		{10 * EB, 10},
		{0.5 * EB, 0.5},
		{0.001 * EB, 0.001},
	}

	for _, tt := range table {
		assert.Equal(t, tt.valueExpected, tt.value.Exabytes())
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
		{KB, "1KB"},
		{MB, "1MB"},
		{GB, "1GB"},
		{TB, "1TB"},
		{PB, "1PB"},
		{EB, "1EB"},
		{400 * TB, "400TB"},
		{2048 * MB, "2GB"},
		{B + KB, "1025B"},
		{MB + 20*KB, "1044KB"},
		{100*MB + KB, "102401KB"},
	}

	for _, tt := range table {
		assert.Equal(t, tt.valueString, tt.value.String())
	}
}
