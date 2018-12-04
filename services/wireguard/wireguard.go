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

	"github.com/mysteriumnetwork/node/money"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
)

// ServiceType indicates "wireguard" service type
const ServiceType = "wireguard"

// ServiceDefinition structure represents "wireguard" service parameters
type ServiceDefinition struct {
	// Approximate information on location where the service is provided from
	Location dto_discovery.Location `json:"location"`
}

// GetLocation returns geographic location of service definition provider
func (service ServiceDefinition) GetLocation() dto_discovery.Location {
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
	Config() (ServiceConfig, error)
	Stop() error
}

// ServiceConfig represent a Wireguard service provider configuration that will be passed to the consumer for establishing a connection.
type ServiceConfig struct {
	Provider struct {
		PublicKey string
		Endpoint  net.UDPAddr
	}
	Consumer struct {
		PrivateKey string // TODO peer private key should be generated on consumer side
	}
	Subnet net.IPNet
}

// MarshalJSON implements json.Marshaler interface to provide human readable configuration.
func (s ServiceConfig) MarshalJSON() ([]byte, error) {
	type provider struct {
		PublicKey string `json:"public_key"`
		Endpoint  string `json:"endpoint"`
	}
	type consumer struct {
		PrivateKey string `json:"private_key"`
	}

	return json.Marshal(&struct {
		Provider provider `json:"provider"`
		Consumer consumer `json:"consumer"`
		Subnet   string   `json:"subnet"`
	}{
		provider{
			s.Provider.PublicKey,
			s.Provider.Endpoint.String(),
		},
		consumer{
			s.Consumer.PrivateKey,
		},
		s.Subnet.String(),
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
	}
	var config struct {
		Provider provider `json:"provider"`
		Consumer consumer `json:"consumer"`
		Subnet   string   `json:"subnet"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	endpoint, err := net.ResolveUDPAddr("udp", config.Provider.Endpoint)
	if err != nil {
		return err
	}
	_, subnet, err := net.ParseCIDR(config.Subnet)
	if err != nil {
		return err
	}

	s.Provider.PublicKey = config.Provider.PublicKey
	s.Consumer.PrivateKey = config.Consumer.PrivateKey
	s.Provider.Endpoint = *endpoint
	s.Subnet = *subnet

	return nil
}
