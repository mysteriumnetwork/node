/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package stringutil

import (
	"strings"
)

// Split slices s into all substrings separated by sep and returns a slice of
// the substrings between those separators.
//
// Difference from the stdlib strings.Split():
// If s does not contain sep, it returns a nil slice.
func Split(s string, sep rune) []string {
	res := strings.FieldsFunc(s, func(c rune) bool { return c == sep })
	if len(res) == 0 {
		return nil
	}
	return res
}
