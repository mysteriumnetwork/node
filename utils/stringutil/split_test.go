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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplit(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		s      string
		sep    rune
		expect []string
	}{

		{s: "11,22,", sep: ',', expect: []string{"11", "22"}},
		{s: "11,", sep: ',', expect: []string{"11"}},
		{s: "", sep: ',', expect: nil},
		{s: "aaaa", sep: ' ', expect: []string{"aaaa"}},
		{s: "a a a a", sep: ' ', expect: []string{"a", "a", "a", "a"}},
	}

	for _, tt := range tests {
		result := Split(tt.s, tt.sep)
		assert.Equal(tt.expect, result)
	}
}
