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

package consumer

import "fmt"

// BitCountDecimal returns a human readable representation of speed in bits per second
// Taken from: https://programming.guide/go/formatting-byte-size-to-human-readable-format.html
func BitCountDecimal(b uint64, suffix string) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d %v", b, suffix)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %c%v", float64(b)/float64(div), "kMGTPE"[exp], suffix)
}
