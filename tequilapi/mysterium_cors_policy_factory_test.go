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

package tequilapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMysteriumCorsPolicy_ReturnsOnlyTrustedOrigins(t *testing.T) {
	policy := NewMysteriumCorsPolicy()
	tests := []struct {
		requested       string
		expectedAllowed string
	}{
		{
			requested:       "http://localhost",
			expectedAllowed: "http://localhost",
		},
		{
			requested:       "http://localhost:8090",
			expectedAllowed: "http://localhost:8090",
		},
		{
			requested:       "http://localhost.evil",
			expectedAllowed: "https://mysterium.network",
		},
		{
			requested:       "https://wallet.mysterium.network",
			expectedAllowed: "https://wallet.mysterium.network",
		},
		{
			requested:       "https://wallet-dev.mysterium.network",
			expectedAllowed: "https://wallet-dev.mysterium.network",
		},
		{
			requested:       "https://wallet-dev.mysterium.network.fake",
			expectedAllowed: "https://mysterium.network",
		},
		{
			requested:       "https://wallet.mysteriumanetwork",
			expectedAllowed: "https://mysterium.network",
		},
		{
			requested:       "http://some-bad-people.com",
			expectedAllowed: "https://mysterium.network",
		},
	}
	for _, tt := range tests {
		allowed := policy.AllowedOrigin(tt.requested)
		assert.Equal(t, tt.expectedAllowed, allowed)
	}
}
