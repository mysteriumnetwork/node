/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package proxyclient

import (
	"net"
	"testing"

	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
	"github.com/stretchr/testify/assert"
)

func Test_ConfigureDevice_ConfigureErrors(t *testing.T) {

	client, err := New()
	assert.NoError(t, err)

	tests := []struct {
		name     string
		config   wgcfg.DeviceConfig
		expected string
	}{
		{
			name:     "empty config",
			config:   wgcfg.DeviceConfig{},
			expected: "could not parse local addr",
		},
		{
			name: "DNS list not provided",
			config: wgcfg.DeviceConfig{
				Subnet: net.IPNet{IP: net.ParseIP("10.0.182.2"), Mask: net.IPv4Mask(255, 255, 255, 0)},
				DNS:    []string{},
			},
			expected: "DNS addr list is empty",
		},
		{
			name: "DNS list contain empty value",
			config: wgcfg.DeviceConfig{
				Subnet: net.IPNet{IP: net.ParseIP("10.0.182.2"), Mask: net.IPv4Mask(255, 255, 255, 0)},
				DNS:    []string{""},
			},
			expected: "could not parse DNS addr",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.ErrorContains(t, client.ConfigureDevice(test.config), test.expected)
		})
	}
}
