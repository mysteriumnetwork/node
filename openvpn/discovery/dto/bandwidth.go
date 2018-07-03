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

package dto

import (
	"github.com/mysterium/node/datasize"
	"strconv"
)

// Bandwidth represents connection bandwidth data type
// Speed in b/s (bits per second)
// 64 b/s = 8 B/s (since there are 8 bits in a byte)
type Bandwidth datasize.BitSize

// MarshalJSON serializes Bandwidth value to JSON compatible value
func (value Bandwidth) MarshalJSON() ([]byte, error) {
	valueBits := datasize.BitSize(value).Bits()
	valueJSON := strconv.FormatUint(valueBits, 10)

	return []byte(valueJSON), nil
}

// UnmarshalJSON restores Bandwidth value from JSON compatible value
func (value *Bandwidth) UnmarshalJSON(valueJSON []byte) error {
	valueBits, err := strconv.ParseUint(string(valueJSON), 10, 64)
	*value = Bandwidth(valueBits)

	return err
}
