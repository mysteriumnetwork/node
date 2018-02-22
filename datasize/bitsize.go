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

	// Kilobyte represents 1024 bytes
	Kilobyte = 1024 * Byte
	// KB is short for Kilobyte
	KB = Kilobyte

	// Megabyte represents 1024 kilobytes
	Megabyte = 1024 * Kilobyte
	// MB is short for Megabyte
	MB = Megabyte

	// Gigabyte represents 1024 megabytes
	Gigabyte = 1024 * Megabyte
	// GB is short for Gigabyte
	GB = Gigabyte

	// Terabyte represents 1024 gigabytes
	Terabyte = 1024 * Gigabyte
	// TB is short for Terabyte
	TB = Terabyte

	// Petabyte represents 1024 terabytes
	Petabyte = 1024 * Terabyte
	// PB is short for Petabyte
	PB = Petabyte

	// Exabyte represents 1024 petabytes
	Exabyte = 1024 * Petabyte
	// EB is short for Exabyte
	EB = Exabyte
)

// Bits returns size in bits
func (size BitSize) Bits() uint64 {
	return uint64(size)
}

// Bytes returns size in bytes
func (size BitSize) Bytes() float64 {
	return float64(size / Byte)
}

// Kilobytes returns size in kilobytes
func (size BitSize) Kilobytes() float64 {
	return float64(size / Kilobyte)
}

// Megabytes returns size in megabytes
func (size BitSize) Megabytes() float64 {
	return float64(size / Megabyte)
}

// Gigabytes returns size in gigabytes
func (size BitSize) Gigabytes() float64 {
	return float64(size / Gigabyte)
}

// Terabytes returns size in terabytes
func (size BitSize) Terabytes() float64 {
	return float64(size / Terabyte)
}

// Petabytes returns size in petabytes
func (size BitSize) Petabytes() float64 {
	return float64(size / Petabyte)
}

// Exabytes returns size in exabytes
func (size BitSize) Exabytes() float64 {
	return float64(size / Exabyte)
}

// String returns human-readable string representation of size
func (size BitSize) String() string {
	switch {
	case size == 0:
		return fmt.Sprintf("%db", size.Bits())

	case size.isDivisible(EB):
		return fmt.Sprintf("%.0fEB", size.Exabytes())

	case size.isDivisible(PB):
		return fmt.Sprintf("%.0fPB", size.Petabytes())

	case size.isDivisible(TB):
		return fmt.Sprintf("%.0fTB", size.Terabytes())

	case size.isDivisible(GB):
		return fmt.Sprintf("%.0fGB", size.Gigabytes())

	case size.isDivisible(MB):
		return fmt.Sprintf("%.0fMB", size.Megabytes())

	case size.isDivisible(KB):
		return fmt.Sprintf("%.0fKB", size.Kilobytes())

	case size.isDivisible(B):
		return fmt.Sprintf("%.0fB", size.Bytes())

	default:
		return fmt.Sprintf("%db", size.Bits())
	}
}

func (size BitSize) isDivisible(divider BitSize) bool {
	return size.Bits()%divider.Bits() == 0
}
