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

import "testing"

func TestRemoveErrorsAndBOMUTF8(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
	}{
		{
			name: "removes BOM",
			args: string([]rune{'\uFEFF', '1', '2'}),
			want: "12",
		},
		{
			name: "removes error runes",
			args: string([]rune{'1', '2', '\uFFFD'}),
			want: "12",
		},
		{
			name: "doesn't change legitimate strings",
			args: string([]rune{'1', '2'}),
			want: "12",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveErrorsAndBOMUTF8(tt.args); got != tt.want {
				t.Errorf("RemoveErrorsAndBOMUTF8() = %v, want %v", got, tt.want)
			}
		})
	}
}
