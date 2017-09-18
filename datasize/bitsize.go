package datasize

import (
	"fmt"
)

type BitSize float64

const (
	Bit BitSize = 1
	b           = Bit

	Byte = 8 * Bit
	B    = Byte

	Kilobyte = 1024 * Byte
	KB       = Kilobyte

	Megabyte = 1024 * Kilobyte
	MB       = Megabyte

	Gigabyte = 1024 * Megabyte
	GB       = Gigabyte

	Terabyte = 1024 * Gigabyte
	TB       = Terabyte

	Petabyte = 1024 * Terabyte
	PB       = Petabyte

	Exabyte = 1024 * Petabyte
	EB      = Exabyte
)

func (size BitSize) Bits() uint64 {
	return uint64(size)
}

func (size BitSize) Bytes() float64 {
	return float64(size / Byte)
}

func (size BitSize) Kilobytes() float64 {
	return float64(size / Kilobyte)
}

func (size BitSize) Megabytes() float64 {
	return float64(size / Megabyte)
}

func (size BitSize) Gigabytes() float64 {
	return float64(size / Gigabyte)
}

func (size BitSize) Terabytes() float64 {
	return float64(size / Terabyte)
}

func (size BitSize) Petabytes() float64 {
	return float64(size / Petabyte)
}

func (size BitSize) Exabytes() float64 {
	return float64(size / Exabyte)
}

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
