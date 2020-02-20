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
	"fmt"
)

// BitSize represents data size in various units.
type BitSize float64

const (
	// Bit represents 1 bit.
	Bit BitSize = 1
	// B is short for Byte.
	B = 8 * Bit
	// KiB is short for Kibibyte.
	KiB = 1024 * B
	// MiB is short for Mebibyte.
	MiB = 1024 * KiB
	// GiB is short for Gibibyte.
	GiB = 1024 * MiB
	// TiB is short for Tebibyte.
	TiB = 1024 * GiB
	// PiB is short for Pebibyte.
	PiB = 1024 * TiB
	// EiB is short for Exbibyte.
	EiB = 1024 * PiB
)

var units = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}

// FromBytes creates BitSize from bytes value.
func FromBytes(bytes uint64) BitSize {
	return BitSize(bytes * B.Bits())
}

// Bits returns size in bits.
func (size BitSize) Bits() uint64 {
	return uint64(size)
}

// Bytes returns size in bytes.
func (size BitSize) Bytes() uint64 {
	return uint64(float64(size) / float64(B.Bits()))
}

// String returns a human readable representation of bytes.
func (size BitSize) String() string {
	if size < B {
		// No fraction on bits
		return fmt.Sprintf("%d %v", uint64(size), "b")
	}
	if size < KiB {
		// No fraction on bytes
		return fmt.Sprintf("%d %v", size.Bytes(), "B")
	}
	val := float64(size.Bytes())
	i := 0
	maxUnit := len(units) - 1
	for val >= 1024 && i < maxUnit {
		val = val / 1024
		i++
	}
	return fmt.Sprintf("%.1f %s", val, units[i])
}
