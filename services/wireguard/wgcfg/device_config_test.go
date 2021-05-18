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

package wgcfg

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviceConfig_Encode(t *testing.T) {
	tests := []struct {
		name     string
		config   DeviceConfig
		expected string
	}{
		{
			name: "Test encode all filled values",
			config: DeviceConfig{
				PrivateKey: "DyxwLJ++jVO+azusu7rPEnzdgfm+0fiOBQ1GTbkk3QQ=",
				ListenPort: 53511,
				Peer: Peer{
					PublicKey:              "DyxwLJ++jVO+azusu7rPEnzdgfm+0fiOBQ1GTbkk3QQ=",
					Endpoint:               endpoint(),
					AllowedIPs:             []string{"192.168.4.10/32", "192.168.4.11/32"},
					KeepAlivePeriodSeconds: 20,
				},
			},
			expected: `private_key=0f2c702c9fbe8d53be6b3bacbbbacf127cdd81f9bed1f88e050d464db924dd04
listen_port=53511
replace_peers=false
public_key=0f2c702c9fbe8d53be6b3bacbbbacf127cdd81f9bed1f88e050d464db924dd04
persistent_keepalive_interval=20
endpoint=182.122.22.19:3233
allowed_ip=192.168.4.10/32
allowed_ip=192.168.4.11/32
`,
		},
		{
			name: "Test encode default values",
			config: DeviceConfig{
				PrivateKey: "",
				ListenPort: 0,
			},
			expected: `private_key=
listen_port=0
replace_peers=false
public_key=
persistent_keepalive_interval=0
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.config.Encode())
		})
	}
}

func TestDeviceConfig_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		config   DeviceConfig
		expected string
	}{
		{
			name: "Test marshal all filled values",
			config: DeviceConfig{
				IfaceName:    "myst0",
				Subnet:       net.IPNet{IP: net.ParseIP("10.0.182.2"), Mask: net.IPv4Mask(255, 255, 255, 0)},
				PrivateKey:   "DyxwLJ++jVO+azusu7rPEnzdgfm+0fiOBQ1GTbkk3QQ=",
				ListenPort:   53511,
				DNS:          []string{"1.1.1.1"},
				DNSScriptDir: "/etc/resolv.conf",
				Peer: Peer{
					PublicKey:              "DyxwLJ++jVO+azusu7rPEnzdgfm+0fiOBQ1GTbkk3QQ=",
					Endpoint:               endpoint(),
					AllowedIPs:             []string{"192.168.4.10/32", "192.168.4.11/32"},
					KeepAlivePeriodSeconds: 20,
				},
			},
			expected: `{"iface_name":"myst0","subnet":"10.0.182.2/24","private_key":"DyxwLJ++jVO+azusu7rPEnzdgfm+0fiOBQ1GTbkk3QQ=","listen_port":53511,"dns":["1.1.1.1"],"dns_script_dir":"/etc/resolv.conf","peer":{"public_key":"DyxwLJ++jVO+azusu7rPEnzdgfm+0fiOBQ1GTbkk3QQ=","endpoint":"182.122.22.19:3233","allowed_i_ps":["192.168.4.10/32","192.168.4.11/32"],"keep_alive_period_seconds":20},"replace_peers":false}`,
		},
		{
			name: "Test marshal default values",
			config: DeviceConfig{
				PrivateKey: "",
				IfaceName:  "",
				Subnet: net.IPNet{
					IP:   nil,
					Mask: nil,
				},
				ListenPort:   0,
				DNS:          []string{},
				DNSScriptDir: "",
				Peer: Peer{
					PublicKey:              "",
					Endpoint:               nil,
					AllowedIPs:             nil,
					KeepAlivePeriodSeconds: 0,
				},
			},
			expected: `{"iface_name":"","subnet":"\u003cnil\u003e","private_key":"","listen_port":0,"dns":[],"dns_script_dir":"","peer":{"public_key":"","endpoint":"","allowed_i_ps":null,"keep_alive_period_seconds":0},"replace_peers":false}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualCfg, err := json.Marshal(test.config)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, string(actualCfg))
		})
	}
}

func TestDeviceConfig_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		expected DeviceConfig
		config   string
	}{
		{
			name:   "Test unmarshal all filled values",
			config: `{"iface_name":"myst0","subnet":"10.0.182.2/24","private_key":"DyxwLJ++jVO+azusu7rPEnzdgfm+0fiOBQ1GTbkk3QQ=","listen_port":53511,"peer":{"public_key":"DyxwLJ++jVO+azusu7rPEnzdgfm+0fiOBQ1GTbkk3QQ=","endpoint":"182.122.22.19:3233","allowed_i_ps":["192.168.4.10/32","192.168.4.11/32"],"keep_alive_period_seconds":20}}`,
			expected: DeviceConfig{
				IfaceName:  "myst0",
				Subnet:     net.IPNet{IP: net.ParseIP("10.0.182.2"), Mask: net.IPv4Mask(255, 255, 255, 0)},
				PrivateKey: "DyxwLJ++jVO+azusu7rPEnzdgfm+0fiOBQ1GTbkk3QQ=",
				ListenPort: 53511,
				Peer: Peer{
					PublicKey:              "DyxwLJ++jVO+azusu7rPEnzdgfm+0fiOBQ1GTbkk3QQ=",
					Endpoint:               endpoint(),
					AllowedIPs:             []string{"192.168.4.10/32", "192.168.4.11/32"},
					KeepAlivePeriodSeconds: 20,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualCfg := DeviceConfig{}
			err := json.Unmarshal([]byte(test.config), &actualCfg)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, actualCfg)
		})
	}
}

func endpoint() *net.UDPAddr {
	res, _ := net.ResolveUDPAddr("udp", "182.122.22.19:3233")
	return res
}
