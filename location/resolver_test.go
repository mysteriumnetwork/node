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

package location

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestResolverResolveCountry(t *testing.T) {
	tests := []struct {
		ip      string
		want    string
		wantErr string
	}{
		{"8.8.8.8", "US", ""},
		{"8.8.4.4", "US", ""},
		{"95.85.39.36", "NL", ""},
		{"127.0.0.1", "", "failed to resolve country"},
		{"8.8.8.8.8", "", "failed to parse IP"},
		{"185.243.112.225", "", "failed to resolve country"},
		{"asd", "", "failed to parse IP"},
	}

	resolver := NewResolver("../bin/common_package/GeoLite2-Country.mmdb")
	for _, tt := range tests {
		got, err := resolver.ResolveCountry(tt.ip)

		assert.Equal(t, tt.want, got, tt.ip)
		if tt.wantErr != "" {
			assert.EqualError(t, err, tt.wantErr, tt.ip)
		} else {
			assert.NoError(t, err, tt.ip)
		}
	}
}
