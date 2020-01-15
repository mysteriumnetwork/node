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

package wireguard

import (
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/money"
	"github.com/stretchr/testify/assert"
)

func Test_PaymentMethod_Serialize(t *testing.T) {
	price := money.NewMoney(50000000, money.CurrencyMyst)

	var tests = []struct {
		model        Payment
		expectedJSON string
	}{
		{
			Payment{
				Price: price,
			},
			`{
				"price": {
					"amount": 50000000,
					"currency": "MYST"
				}
			}`,
		},
		{
			Payment{},
			`{
				"price": {}
			}`,
		},
	}

	for _, test := range tests {
		jsonBytes, err := json.Marshal(test.model)

		assert.Nil(t, err)
		assert.JSONEq(t, test.expectedJSON, string(jsonBytes))
	}
}

func Test_PaymentMethod_Unserialize(t *testing.T) {
	price := money.NewMoney(50000000, money.CurrencyMyst)

	var tests = []struct {
		json          string
		expectedModel Payment
		expectedError error
	}{
		{
			`{
				"price": {
					"amount": 50000000,
					"currency": "MYST"
				}
			}`,
			Payment{
				Price: price,
			},
			nil,
		},
		{
			`{
				"price": {}
			}`,
			Payment{},
			nil,
		},
		{
			`{}`,
			Payment{},
			nil,
		},
	}

	for _, test := range tests {
		var model Payment
		err := json.Unmarshal([]byte(test.json), &model)

		assert.Equal(t, test.expectedModel, model)
		assert.Equal(t, test.expectedError, err)
	}
}

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
			},
			expected: `private_key=0f2c702c9fbe8d53be6b3bacbbbacf127cdd81f9bed1f88e050d464db924dd04
listen_port=53511
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
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.config.Encode())
		})
	}
}

func TestPeerConfig_Encode(t *testing.T) {
	endpoint := func() *net.UDPAddr {
		res, _ := net.ResolveUDPAddr("udp", "182.122.22.19:3233")
		return res
	}
	tests := []struct {
		name     string
		peer     Peer
		expected string
	}{
		{
			name: "Test encode all filled values",
			peer: Peer{
				PublicKey:              "DyxwLJ++jVO+azusu7rPEnzdgfm+0fiOBQ1GTbkk3QQ=",
				Endpoint:               endpoint(),
				AllowedIPs:             []string{"192.168.4.10/32", "192.168.4.11/32"},
				KeepAlivePeriodSeconds: 20,
			},
			expected: `public_key=0f2c702c9fbe8d53be6b3bacbbbacf127cdd81f9bed1f88e050d464db924dd04
persistent_keepalive_interval=20
endpoint=182.122.22.19:3233
allowed_ip=192.168.4.10/32
allowed_ip=192.168.4.11/32
`,
		},
		{
			name: "Test encode default values",
			peer: Peer{
				PublicKey:              "",
				Endpoint:               nil,
				AllowedIPs:             []string{},
				KeepAlivePeriodSeconds: 0,
			},
			expected: `public_key=
persistent_keepalive_interval=0
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.peer.Encode())
		})
	}
}

func TestParsePeerStats(t *testing.T) {
	tests := []struct {
		name          string
		device        *UserspaceDevice
		expectedStats *Stats
		expectedErr   error
	}{
		{
			name: "Test parse stats successfully",
			device: &UserspaceDevice{
				Peers: []UserspaceDevicePeer{
					{
						TransmitBytes:     10,
						ReceiveBytes:      12,
						LastHandshakeTime: time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC),
					},
				},
			},
			expectedStats: &Stats{
				BytesSent:     10,
				BytesReceived: 12,
				LastHandshake: time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC),
			},
			expectedErr: nil,
		},
		{
			name: "Test parse fail when more than one peer returned",
			device: &UserspaceDevice{
				Peers: []UserspaceDevicePeer{{}, {}},
			},
			expectedStats: nil,
			expectedErr:   fmt.Errorf("exactly 1 peer expected, got %d", 2),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stats, err := ParseDevicePeerStats(test.device)
			assert.Equal(t, test.expectedErr, err)
			assert.Equal(t, test.expectedStats, stats)
		})
	}
}
