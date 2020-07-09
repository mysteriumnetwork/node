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

package session

import (
	"github.com/asdine/storm/v3/q"
)

// Filter defines all flags for session filtering in session storage.
type Filter struct {
	Direction string
}

// Match is used to test the criteria against a structure.
func (f Filter) Match(i interface{}) (bool, error) {
	matchers := make([]q.Matcher, 0)
	if f.Direction != "" {
		matchers = append(matchers, q.Eq("Direction", f.Direction))
	}

	return q.And(matchers...).Match(i)
}
