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
)

// ServiceType indicates "wireguard" service type
const ServiceType = "wireguard"

// ServiceConfig represent a Wireguard service provider configuration that will be passed to the consumer for establishing a connection.
type ServiceConfig struct {
	// LocalPort and RemotePort are needed for NAT hole punching only.
	LocalPort  int   `json:"-"`
	RemotePort int   `json:"-"`
	Ports      []int `json:"ports"`

	Provider struct {
		PublicKey string
		Endpoint  net.UDPAddr
	}
	Consumer struct {
		IPAddress net.IPNet
		DNSIPs    string
	}
}

// ConsumerConfig is used for sending the public key and IP from consumer to provider.
type ConsumerConfig struct {
	PublicKey string `json:"PublicKey"`
	// IP is needed when provider is behind NAT. In such case provider parses this IP and tries to ping consumer.
	IP    string `json:"IP,omitempty"`
	Ports []int  `json:"Ports"`
}

// MarshalJSON implements json.Marshaler interface to provide human readable configuration.
func (s ServiceConfig) MarshalJSON() ([]byte, error) {
	type provider struct {
		PublicKey string `json:"public_key"`
		Endpoint  string `json:"endpoint"`
	}
	type consumer struct {
		IPAddress string `json:"ip_address"`
		DNSIPs    string `json:"dns_ips"`
	}

	return json.Marshal(&struct {
		LocalPort  int      `json:"local_port"`
		RemotePort int      `json:"remote_port"`
		Ports      []int    `json:"ports"`
		Provider   provider `json:"provider"`
		Consumer   consumer `json:"consumer"`
	}{
		Ports:      s.Ports,
		LocalPort:  s.LocalPort,
		RemotePort: s.RemotePort,
		Provider: provider{
			PublicKey: s.Provider.PublicKey,
			Endpoint:  s.Provider.Endpoint.String(),
		},
		Consumer: consumer{
			IPAddress: s.Consumer.IPAddress.String(),
			DNSIPs:    s.Consumer.DNSIPs,
		},
	})
}

// UnmarshalJSON implements json.Unmarshaler interface to receive human readable configuration.
func (s *ServiceConfig) UnmarshalJSON(data []byte) error {
	type provider struct {
		PublicKey string `json:"public_key"`
		Endpoint  string `json:"endpoint"`
	}
	type consumer struct {
		IPAddress string `json:"ip_address"`
		DNSIPs    string `json:"dns_ips"`
	}
	var config struct {
		LocalPort  int      `json:"local_port"`
		RemotePort int      `json:"remote_port"`
		Ports      []int    `json:"ports"`
		Provider   provider `json:"provider"`
		Consumer   consumer `json:"consumer"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	endpoint, err := net.ResolveUDPAddr("udp", config.Provider.Endpoint)
	if err != nil {
		return err
	}
	ip, ipnet, err := net.ParseCIDR(config.Consumer.IPAddress)
	if err != nil {
		return err
	}

	s.Ports = config.Ports
	s.LocalPort = config.LocalPort
	s.RemotePort = config.RemotePort
	s.Provider.Endpoint = *endpoint
	s.Provider.PublicKey = config.Provider.PublicKey
	s.Consumer.DNSIPs = config.Consumer.DNSIPs
	s.Consumer.IPAddress = *ipnet
	s.Consumer.IPAddress.IP = ip

	return nil
}
