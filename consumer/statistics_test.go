/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

import (
	"reflect"
	"testing"
)

func TestSessionStatistics_DiffWithNew(t *testing.T) {
	exampleStats := SessionStatistics{
		BytesReceived: 1,
		BytesSent:     2,
	}
	tests := []struct {
		name string
		old  SessionStatistics
		new  SessionStatistics
		want SessionStatistics
	}{
		{
			name: "calculates statistics correctly if they are continuous",
			old:  SessionStatistics{},
			new:  exampleStats,
			want: exampleStats,
		},
		{
			name: "calculates statistics correctly if they are not continuous",
			old: SessionStatistics{
				BytesReceived: 5,
				BytesSent:     6,
			},
			new:  exampleStats,
			want: exampleStats,
		},
		{
			name: "returns zeros on no change",
			old:  exampleStats,
			new:  exampleStats,
			want: SessionStatistics{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := tt.old
			if got := ss.DiffWithNew(tt.new); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SessionStatistics.DiffWithNew() = %v, want %v", got, tt.want)
			}
		})
	}
}
