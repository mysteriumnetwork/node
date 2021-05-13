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

package wireguard

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceConfig_MarshalJSON(t *testing.T) {
	endpoint, _ := net.ResolveUDPAddr("udp4", "127.0.0.1:51001")
	config := ServiceConfig{
		LocalPort:  51000,
		RemotePort: 51001,
		Provider: struct {
			PublicKey string
			Endpoint  net.UDPAddr
		}{
			PublicKey: "wg1",
			Endpoint:  *endpoint,
		},
		Consumer: struct {
			IPAddress net.IPNet
			DNSIPs    string
		}{
			IPAddress: net.IPNet{
				IP:   net.IPv4(127, 0, 0, 1),
				Mask: net.IPv4Mask(255, 255, 255, 128),
			},
			DNSIPs: "128.0.0.1",
		},
	}

	configBytes, err := json.Marshal(config)
	assert.NoError(t, err)
	assert.Equal(t,
		`{"local_port":51000,"remote_port":51001,"ports":null,"provider":{"public_key":"wg1","endpoint":"127.0.0.1:51001"},"consumer":{"ip_address":"127.0.0.1/25","dns_ips":"128.0.0.1"}}`,
		string(configBytes),
	)
}

func TestServiceConfig_UnmarshalJSON(t *testing.T) {
	configJSON := json.RawMessage(`{"local_port":51000,"remote_port":51001,"provider":{"public_key":"wg1","endpoint":"127.0.0.1:51001"},"consumer":{"ip_address":"127.0.0.1/25","dns_ips":"128.0.0.1"}}`)

	endpoint, _ := net.ResolveUDPAddr("udp4", "127.0.0.1:51001")
	expecteConfig := ServiceConfig{
		LocalPort:  51000,
		RemotePort: 51001,
		Provider: struct {
			PublicKey string
			Endpoint  net.UDPAddr
		}{
			PublicKey: "wg1",
			Endpoint:  *endpoint,
		},
		Consumer: struct {
			IPAddress net.IPNet
			DNSIPs    string
		}{
			IPAddress: net.IPNet{
				IP:   net.IPv4(127, 0, 0, 1),
				Mask: net.IPv4Mask(255, 255, 255, 128),
			},
			DNSIPs: "128.0.0.1",
		},
	}

	var actualConfig ServiceConfig
	err := json.Unmarshal(configJSON, &actualConfig)

	assert.NoError(t, err)
	assert.Equal(t, expecteConfig, actualConfig)
}
