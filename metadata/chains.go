/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package metadata

// chains holds data about chains
type chains struct {
	l1 []int64
	l2 []int64
}

// ChainData stores static chain data which doesn't change at runtime.
var ChainData = chains{
	l1: []int64{5},
	l2: []int64{80001},
}

// IsL1 returns if a given chains belongs to level 1.
func (c *chains) IsL1(chainID int64) bool {
	for _, id := range c.l1 {
		if id == chainID {
			return true
		}
	}

	return false
}

// IsL1 returns if a given chains belongs to level 2.
func (c *chains) IsL2(chainID int64) bool {
	for _, id := range c.l2 {
		if id == chainID {
			return true
		}
	}

	return false
}
