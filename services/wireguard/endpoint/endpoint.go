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

package endpoint

import (
	"net"

	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/nat"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
)

type wgClient interface {
	ConfigureDevice(name string, config wg.DeviceConfig, subnet net.IPNet) error
	AddPeer(name string, peer wg.PeerInfo) error
	Close() error
}

type connectionEndpoint struct {
	iface             string
	privateKey        string
	subnet            net.IPNet
	endpoint          net.UDPAddr
	ipResolver        ip.Resolver
	resourceAllocator resources.Allocator
	wgClient          wgClient
	natService        nat.NATService
}

// Start starts and configure wireguard network interface for providing service.
// If config is nil, required options will be generated automatically.
func (ce *connectionEndpoint) Start(config *wg.ServiceConfig) error {
	ce.iface = ce.resourceAllocator.AllocateInterface()
	ce.endpoint.Port = ce.resourceAllocator.AllocatePort()

	if config == nil {
		privateKey, err := GeneratePrivateKey()
		if err != nil {
			return err
		}
		ce.subnet = ce.resourceAllocator.AllocateIPNet()
		ce.subnet.IP = providerIP(ce.subnet)
		ce.privateKey = privateKey
	} else {
		ce.subnet = config.Subnet
		ce.subnet.IP = consumerIP(ce.subnet)
		ce.privateKey = config.Consumer.PrivateKey
	}

	if ce.ipResolver != nil {
		publicIP, err := ce.ipResolver.GetPublicIP()
		if err != nil {
			return err
		}
		ce.endpoint.IP = net.ParseIP(publicIP)

		outboundIP, err := ce.ipResolver.GetOutboundIP()
		if err != nil {
			return err
		}

		ce.natService = nat.NewService()
		ce.natService.Add(nat.RuleForwarding{
			SourceAddress: ce.subnet.String(),
			TargetIP:      outboundIP,
		})

		if err := ce.natService.Start(); err != nil {
			return err
		}
	}

	var deviceConfig deviceConfig
	deviceConfig.listenPort = ce.endpoint.Port
	deviceConfig.privateKey = ce.privateKey
	return ce.wgClient.ConfigureDevice(ce.iface, deviceConfig, ce.subnet)
}

// AddPeer adds new wireguard peer to the wireguard network interface.
func (ce *connectionEndpoint) AddPeer(publicKey string, endpoint *net.UDPAddr) error {
	return ce.wgClient.AddPeer(ce.iface, peerInfo{endpoint, publicKey})
}

// Config provides wireguard service configuration for the current connection endpoint.
func (ce *connectionEndpoint) Config() (wg.ServiceConfig, error) {
	publicKey, err := PrivateKeyToPublicKey(ce.privateKey)
	if err != nil {
		return wg.ServiceConfig{}, err
	}

	var config wg.ServiceConfig
	config.Provider.PublicKey = publicKey
	config.Provider.Endpoint = ce.endpoint
	config.Subnet = ce.subnet
	return config, nil
}

// Stop closes wireguard client and destroys wireguard network interface.
func (ce *connectionEndpoint) Stop() error {
	if err := ce.wgClient.Close(); err != nil {
		return err
	}

	if ce.natService != nil {
		ce.natService.Stop()
	}

	if err := ce.resourceAllocator.ReleasePort(ce.endpoint.Port); err != nil {
		return err
	}

	if err := ce.resourceAllocator.ReleaseIPNet(ce.subnet); err != nil {
		return err
	}

	return ce.resourceAllocator.ReleaseInterface(ce.iface)
}

type deviceConfig struct {
	privateKey string
	listenPort int
}

func (d deviceConfig) PrivateKey() string {
	return d.privateKey
}

func (d deviceConfig) ListenPort() int {
	return d.listenPort
}

type peerInfo struct {
	endpoint  *net.UDPAddr
	publicKey string
}

func (p peerInfo) Endpoint() *net.UDPAddr {
	return p.endpoint
}
func (p peerInfo) PublicKey() string {
	return p.publicKey
}

func providerIP(subnet net.IPNet) net.IP {
	subnet.IP[len(subnet.IP)-1] = byte(1)
	return subnet.IP
}

func consumerIP(subnet net.IPNet) net.IP {
	subnet.IP[len(subnet.IP)-1] = byte(2)
	return subnet.IP
}
