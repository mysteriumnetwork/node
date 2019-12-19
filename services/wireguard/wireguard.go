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
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
)

// ServiceType indicates "wireguard" service type
const ServiceType = "wireguard"

// ServiceDefinition structure represents "wireguard" service parameters
type ServiceDefinition struct {
	// Approximate information on location where the service is provided from
	Location market.Location `json:"location"`

	// Approximate information on location where the actual tunnelled traffic will originate from.
	// This is used by providers having their own means of setting tunnels to other remote exit points.
	LocationOriginate market.Location `json:"location_originate"`
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
	AddPeer(iface string, peer AddPeerOptions) error
	PeerStats() (*Stats, error)
	ConfigureRoutes(ip net.IP) error
	Config() (ServiceConfig, error)
	InterfaceName() string
	Stop() error
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
		PrivateKey   string `json:"-"`
		IPAddress    net.IPNet
		DNSIPs       string
		ConnectDelay int
	}
}

// MarshalJSON implements json.Marshaler interface to provide human readable configuration.
func (s ServiceConfig) MarshalJSON() ([]byte, error) {
	type provider struct {
		PublicKey string `json:"public_key"`
		Endpoint  string `json:"endpoint"`
	}
	type consumer struct {
		PrivateKey   string `json:"private_key"`
		IPAddress    string `json:"ip_address"`
		DNSIPs       string `json:"dns_ips"`
		ConnectDelay int    `json:"connect_delay"`
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
			IPAddress:    s.Consumer.IPAddress.String(),
			ConnectDelay: s.Consumer.ConnectDelay,
			DNSIPs:       s.Consumer.DNSIPs,
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
		PrivateKey   string `json:"private_key"`
		IPAddress    string `json:"ip_address"`
		DNSIPs       string `json:"dns_ips"`
		ConnectDelay int    `json:"connect_delay"`
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
	s.Consumer.DNSIPs = config.Consumer.DNSIPs
	s.Consumer.IPAddress = *ipnet
	s.Consumer.IPAddress.IP = ip
	s.Consumer.ConnectDelay = config.Consumer.ConnectDelay

	return nil
}

// DeviceConfig describes wireguard device configuration.
type DeviceConfig struct {
	PrivateKey string
	ListenPort int
}

// Encode encodes device config into string representation which is used for
// userspace and kernel space wireguard configuration.
func (dc *DeviceConfig) Encode() string {
	var res strings.Builder
	keyBytes, err := base64.StdEncoding.DecodeString(dc.PrivateKey)
	if err != nil {
		return ""
	}
	hexKey := hex.EncodeToString(keyBytes)

	res.WriteString(fmt.Sprintf("private_key=%s\n", hexKey))
	res.WriteString(fmt.Sprintf("listen_port=%d\n", dc.ListenPort))
	return res.String()
}

// AddPeerOptions represents wireguard new peer options.
type AddPeerOptions struct {
	Endpoint  *net.UDPAddr
	PublicKey string
}

// Peer represents wireguard peer.
type Peer struct {
	PublicKey       string
	Endpoint        *net.UDPAddr
	AllowedIPs      []string
	KeepAlivePeriod int
}

// Encode encodes device peer config into string representation which is used for
// userspace and kernel space wireguard configuration.
func (p *Peer) Encode() string {
	var res strings.Builder

	keyBytes, err := base64.StdEncoding.DecodeString(p.PublicKey)
	if err != nil {
		return ""
	}
	hexKey := hex.EncodeToString(keyBytes)
	res.WriteString(fmt.Sprintf("public_key=%s\n", hexKey))
	res.WriteString(fmt.Sprintf("persistent_keepalive_interval=%d\n", p.KeepAlivePeriod))
	if p.Endpoint != nil {
		res.WriteString(fmt.Sprintf("endpoint=%s\n", p.Endpoint.String()))
	}
	if len(p.AllowedIPs) > 0 {
		for _, ip := range p.AllowedIPs {
			res.WriteString(fmt.Sprintf("allowed_ip=%s\n", ip))
		}
	}
	return res.String()
}

// Stats represents wireguard peer statistics information.
type Stats struct {
	BytesSent     uint64
	BytesReceived uint64
	LastHandshake time.Time
}

// ParseDevicePeerStats parses current active consumer stats.
func ParseDevicePeerStats(d *UserspaceDevice) (*Stats, error) {
	if len(d.Peers) != 1 {
		return nil, fmt.Errorf("exactly 1 peer expected, got %d", len(d.Peers))
	}

	p := d.Peers[0]
	return &Stats{
		BytesSent:     uint64(p.TransmitBytes),
		BytesReceived: uint64(p.ReceiveBytes),
		LastHandshake: p.LastHandshakeTime,
	}, nil
}
