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

type connectionEndpoint struct {
	iface             string
	ip                net.IPNet
	endpoint          net.UDPAddr
	wgClient          *wireguardctrl.Client
	ipResolver        ip.Resolver
	resourceAllocator resources.Allocator
}

// NewConnectionEndpoint creates Wireguard client with predefined interface name and endpoint.
func NewConnectionEndpoint(ipResolver ip.Resolver) (*connectionEndpoint, error) {
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
func (ce *connectionEndpoint) Start() error {
	publicIP, err := ce.ipResolver.GetPublicIP()
	if err != nil {
		return err
	}

	ce.ip = ce.resourceAllocator.AllocateIP()
	ce.iface = ce.resourceAllocator.AllocateInterface()
	ce.endpoint = net.UDPAddr{IP: net.ParseIP(publicIP), Port: ce.resourceAllocator.AllocatePort()}
	if err := ce.up(); err != nil {
		return err
	}

	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return err
	}

	config := wgtypes.Config{
		PrivateKey:   &key,
		ListenPort:   &ce.endpoint.Port,
		ReplacePeers: true,
	}

	if err := ce.wgClient.ConfigureDevice(ce.iface, config); err != nil {
		return err
	}

	return nil
}

// NewConsumer adds service consumer public key to the list of allowed peers.
func (ce *connectionEndpoint) NewConsumer() (configProvider, error) {
	peerKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return consumer{}, err
	}

	ip := ce.resourceAllocator.AllocateIP()
	config := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{{
			Endpoint:   &ce.endpoint,
			PublicKey:  peerKey.PublicKey(),
			AllowedIPs: []net.IPNet{ip},
		}},
	}

	if err := ce.wgClient.ConfigureDevice(ce.iface, config); err != nil {
		return consumer{}, err
	}

	return consumer{
		interfaceAddress: ip,
		peer:             config.Peers[0],
		privateKey:       peerKey,
	}, nil
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

	if err := ce.resourceAllocator.ReleaseIP(ce.ip); err != nil {
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

	if err := exec.Command("ip", "address", "replace", "dev", ce.iface, ce.ip.String()).Run(); err != nil {
		return err
	}

	return exec.Command("ip", "link", "set", "dev", ce.iface, "up").Run()
}
