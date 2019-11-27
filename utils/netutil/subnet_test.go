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

package netutil

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFirstIP(t *testing.T) {
	tests := []struct {
		name string
		args net.IPNet
		want net.IP
	}{
		{
			name: "Normal case",
			args: net.IPNet{IP: net.ParseIP("1.2.3.4").To4(), Mask: net.IPv4Mask(255, 255, 255, 0)},
			want: net.ParseIP("1.2.3.1").To4(),
		},
		{
			name: "Normal case non /24",
			args: net.IPNet{IP: net.ParseIP("1.2.3.218").To4(), Mask: net.IPv4Mask(255, 255, 255, 128)},
			want: net.ParseIP("1.2.3.129").To4(),
		},
		{
			name: "Normal case /32",
			args: net.IPNet{IP: net.ParseIP("1.2.3.218").To4(), Mask: net.IPv4Mask(255, 255, 255, 255)},
			want: net.ParseIP("1.2.3.218").To4(),
		},
		{
			name: "Empty subnet not panic",
			args: net.IPNet{},
			want: net.IP{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FirstIP(tt.args))
		})
	}
}
