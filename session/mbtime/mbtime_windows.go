/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package mbtime

import (
	// Needed for go:linkname
	_ "unsafe"
)

//go:noescape
//go:linkname nanotimeWin runtime.nanotime
func nanotimeWin() int64

func nanotime() (uint64, error) {
	// On Windows nanotime should already include boot time so we can link it from runtime.
	return uint64(nanotimeWin()), nil
}
