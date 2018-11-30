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
	"net"
	"os/exec"

	"github.com/mdlayher/wireguardctrl"
	"github.com/mdlayher/wireguardctrl/wgtypes"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
)

var allowedIPs = []net.IPNet{
	{IP: net.IPv4zero, Mask: net.CIDRMask(0, 32)},
	{IP: net.IPv6zero, Mask: net.CIDRMask(0, 128)},
}

type connectionEndpoint struct {
	iface             string
	subnet            net.IPNet
	endpoint          net.UDPAddr
	ipResolver        ip.Resolver
	resourceAllocator resources.Allocator
	wgClient          *wireguardctrl.Client
}

// NewConnectionEndpoint creates new wireguard connection endpoint.
func NewConnectionEndpoint(ipResolver ip.Resolver) (ConnectionEndpoint, error) {
	wgClient, err := wireguardctrl.New()
	if err != nil {
		return nil, err
	}

	return &connectionEndpoint{
		wgClient:          wgClient,
		ipResolver:        ipResolver,
		resourceAllocator: resources.Allocator{},
	}, nil
}

// Start starts and configure wireguard network interface for providing service.
// If config is nil, required options will be generated automatically.
func (ce *connectionEndpoint) Start(config *serviceConfig) error {
	ce.iface = ce.resourceAllocator.AllocateInterface()
	ce.endpoint.Port = ce.resourceAllocator.AllocatePort()
	if ce.ipResolver != nil {
		publicIP, err := ce.ipResolver.GetPublicIP()
		if err != nil {
			return err
		}
		ce.endpoint.IP = net.ParseIP(publicIP)
	}

	deviceConfig := wgtypes.Config{
		ReplacePeers: true,
		ListenPort:   &ce.endpoint.Port,
	}

	if config == nil {
		privateKey, err := wgtypes.GeneratePrivateKey()
		if err != nil {
			return err
		}
		ce.subnet = ce.resourceAllocator.AllocateIPNet()
		ce.subnet.IP = providerIP(ce.subnet)
		deviceConfig.PrivateKey = &privateKey
	} else {
		ce.subnet = config.Subnet
		ce.subnet.IP = consumerIP(ce.subnet)
		deviceConfig.PrivateKey = &config.Consumer.PrivateKey
	}

	if err := ce.up(); err != nil {
		return err
	}

	return ce.wgClient.ConfigureDevice(ce.iface, deviceConfig)
}

// AddPeer adds new wireguard peer to the wireguard network interface.
func (ce *connectionEndpoint) AddPeer(publicKey wgtypes.Key, endpoint *net.UDPAddr) error {
	deviceConfig := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{{
			Endpoint:   endpoint,
			PublicKey:  publicKey,
			AllowedIPs: allowedIPs,
		}},
	}

	return ce.wgClient.ConfigureDevice(ce.iface, deviceConfig)
}

// Config provides wireguard service configuration for the current connection endpoint.
func (ce *connectionEndpoint) Config() (serviceConfig, error) {
	d, err := ce.wgClient.Device(ce.iface)
	if err != nil || d.Name != ce.iface {
		return serviceConfig{}, err
	}

	var config serviceConfig
	config.Provider.PublicKey = d.PublicKey
	config.Provider.Endpoint = ce.endpoint
	config.Subnet = ce.subnet

	return config, nil
}

// Stop closes wireguard client and destroys wireguard network interface.
func (ce *connectionEndpoint) Stop() error {
	if err := ce.wgClient.Close(); err != nil {
		return err
	}

	if err := exec.Command("ip", "link", "del", "dev", ce.iface).Run(); err != nil {
		return err
	}

	if err := ce.resourceAllocator.ReleasePort(ce.endpoint.Port); err != nil {
		return err
	}

	if err := ce.resourceAllocator.ReleaseIPNet(ce.subnet); err != nil {
		return err
	}

	return ce.resourceAllocator.ReleaseInterface(ce.iface)
}

func (ce *connectionEndpoint) up() error {
	if d, err := ce.wgClient.Device(ce.iface); err != nil || d.Name != ce.iface {
		if err := exec.Command("ip", "link", "add", "dev", ce.iface, "type", "wireguard").Run(); err != nil {
			return err
		}
	}

	if err := exec.Command("ip", "address", "replace", "dev", ce.iface, ce.subnet.String()).Run(); err != nil {
		return err
	}

	return exec.Command("ip", "link", "set", "dev", ce.iface, "up").Run()
}

func providerIP(subnet net.IPNet) net.IP {
	subnet.IP[len(subnet.IP)-1] = byte(1)
	return subnet.IP
}

func consumerIP(subnet net.IPNet) net.IP {
	subnet.IP[len(subnet.IP)-1] = byte(2)
	return subnet.IP
}
