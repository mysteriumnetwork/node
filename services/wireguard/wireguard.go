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
	"net"

	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
)

// ServiceType indicates "wireguard" service type
const ServiceType = "wireguard"

// ServiceDefinition structure represents "wireguard" service parameters
type ServiceDefinition struct {
	// Approximate information on location where the service is provided from
	Location market.Location `json:"location"`
}

// GetLocation returns geographic location of service definition provider
func (service ServiceDefinition) GetLocation() market.Location {
	return service.Location
}

// PaymentMethod indicates payment method for Wireguard service
const PaymentMethod = "WG"

// Payment structure describes price for Wireguard service payment
type Payment struct {
	Price money.Money `json:"price"`
}

// GetPrice returns price of payment per time
func (method Payment) GetPrice() money.Money {
	return method.Price
}

// ConnectionEndpoint represents Wireguard network instance, it provide information
// required for establishing connection between service provider and consumer.
type ConnectionEndpoint interface {
	Start(config *ServiceConfig) error
	AddPeer(publicKey string, endpoint *net.UDPAddr) error
	PeerStats() (stats consumer.SessionStatistics, lastHandshake int, err error)
	ConfigureRoutes(ip net.IP) error
	Config() (ServiceConfig, error)
	Stop() error
}

// DeviceConfig describes wireguard device configuration.
type DeviceConfig interface {
	PrivateKey() string
	ListenPort() int
}

// PeerInfo represents wireguard peer information.
type PeerInfo interface {
	Endpoint() *net.UDPAddr
	PublicKey() string
}

// ConsumerConfig is used for sending the public key from consumer to provider
type ConsumerConfig struct {
	PublicKey string
}

// ConsumerPrivateKey represents the private part of the consumer key
type ConsumerPrivateKey struct {
	PrivateKey string
}

// ServiceConfig represent a Wireguard service provider configuration that will be passed to the consumer for establishing a connection.
type ServiceConfig struct {
	Provider struct {
		PublicKey string
		Endpoint  net.UDPAddr
	}
	Consumer struct {
		PrivateKey string `json:"-"`
		IPAddress  net.IPNet
	}
}

// MarshalJSON implements json.Marshaler interface to provide human readable configuration.
func (s ServiceConfig) MarshalJSON() ([]byte, error) {
	type provider struct {
		PublicKey string `json:"public_key"`
		Endpoint  string `json:"endpoint"`
	}
	type consumer struct {
		PrivateKey string `json:"private_key"`
		IPAddress  string `json:"ip_address"`
	}

	return json.Marshal(&struct {
		Provider provider `json:"provider"`
		Consumer consumer `json:"consumer"`
	}{
		provider{
			s.Provider.PublicKey,
			s.Provider.Endpoint.String(),
		},
		consumer{
			IPAddress: s.Consumer.IPAddress.String(),
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
		PrivateKey string `json:"private_key"`
		IPAddress  string `json:"ip_address"`
	}
	var config struct {
		Provider provider `json:"provider"`
		Consumer consumer `json:"consumer"`
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

	s.Provider.Endpoint = *endpoint
	s.Provider.PublicKey = config.Provider.PublicKey
	s.Consumer.IPAddress = *ipnet
	s.Consumer.IPAddress.IP = ip

	return nil
}
