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

// BitSize represents data size in various units
type BitSize float64

const (
	// Bit represents 1 bit
	Bit BitSize = 1
	b           = Bit

	// Byte is 8 bits
	Byte = 8 * Bit
	// B is short for Byte
	B = Byte

	// Kibibyte represents 1024 bytes
	Kibibyte = 1024 * Byte
	// KiB is short for Kibibyte
	KiB = Kibibyte

	// Mebibyte represents 1024 kibibytes
	Mebibyte = 1024 * Kibibyte
	// MiB is short for Mebibyte
	MiB = Mebibyte

	// Gibibyte represents 1024 mebibytes
	Gibibyte = 1024 * Mebibyte
	// GiB is short for Gibibyte
	GiB = Gibibyte

	// Tebibyte represents 1024 gibibytes
	Tebibyte = 1024 * Gibibyte
	// TiB is short for Tebibyte
	TiB = Tebibyte

	// Pebibyte represents 1024 tebibytes
	Pebibyte = 1024 * Tebibyte
	// PiB is short for Pebibyte
	PiB = Pebibyte

	// Exbibyte represents 1024 pebibytes
	Exbibyte = 1024 * Pebibyte
	// EiB is short for Exbibyte
	EiB = Exbibyte
)

// Bits returns size in bits
func (size BitSize) Bits() uint64 {
	return uint64(size)
}

// Bytes returns size in bytes
func (size BitSize) Bytes() float64 {
	return float64(size / Byte)
}

// Kibibytes returns size in kibibytes
func (size BitSize) Kibibytes() float64 {
	return float64(size / Kibibyte)
}

// Mebibytes returns size in mebibytes
func (size BitSize) Mebibytes() float64 {
	return float64(size / Mebibyte)
}

// Gibibytes returns size in gigabytes
func (size BitSize) Gibibytes() float64 {
	return float64(size / Gibibyte)
}

// Tebibytes returns size in tebibytes
func (size BitSize) Tebibytes() float64 {
	return float64(size / Tebibyte)
}

// Pebibytes returns size in pebibytes
func (size BitSize) Pebibytes() float64 {
	return float64(size / Pebibyte)
}

// Exbibytes returns size in exbibytes
func (size BitSize) Exbibytes() float64 {
	return float64(size / Exbibyte)
}

// String returns human-readable string representation of size
func (size BitSize) String() string {
	switch {
	case size == 0:
		return fmt.Sprintf("%db", size.Bits())

	case size.isDivisible(EiB):
		return fmt.Sprintf("%.0fEiB", size.Exbibytes())

	case size.isDivisible(PiB):
		return fmt.Sprintf("%.0fPiB", size.Pebibytes())

	case size.isDivisible(TiB):
		return fmt.Sprintf("%.0fTiB", size.Tebibytes())

	case size.isDivisible(GiB):
		return fmt.Sprintf("%.0fGiB", size.Gibibytes())

	case size.isDivisible(MiB):
		return fmt.Sprintf("%.0fMiB", size.Mebibytes())

	case size.isDivisible(KiB):
		return fmt.Sprintf("%.0fKiB", size.Kibibytes())

	case size.isDivisible(B):
		return fmt.Sprintf("%.0fB", size.Bytes())

	default:
		return fmt.Sprintf("%db", size.Bits())
	}
}

func (size BitSize) isDivisible(divider BitSize) bool {
	return size.Bits()%divider.Bits() == 0
}
