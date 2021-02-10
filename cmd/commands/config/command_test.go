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

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_squishMap(t *testing.T) {
	for _, test := range []struct {
		give map[string]interface{}
		get  map[string]string
	}{
		{
			give: map[string]interface{}{
				"test1": 1,
				"test2": "1",
			},
			get: map[string]string{
				"test1": "1",
				"test2": "1",
			},
		},
		{
			give: map[string]interface{}{
				"test1": map[string]interface{}{
					"test2": 1,
				},
				"test3": "1",
				"test4": map[string]interface{}{
					"test5": 7.5,
				},
			},
			get: map[string]string{
				"test1.test2": "1",
				"test3":       "1",
				"test4.test5": "7.5",
			},
		},
	} {
		dest := map[string]string{}
		squishMap(test.give, dest)
		assert.Len(t, dest, len(test.get))

		for k, v := range test.get {
			got, ok := dest[k]
			assert.True(t, ok)
			assert.Equal(t, v, got)
		}
	}
}
