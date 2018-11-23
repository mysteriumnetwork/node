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

package network

import (
	"fmt"
	"net"
	"os/exec"

	"github.com/mdlayher/wireguardctrl"
	"github.com/mdlayher/wireguardctrl/wgtypes"
)

// NewNetwork creates Wireguard client with predefined interface name and public IP.
func NewNetwork(name, publicIP string) (*network, error) {
	wgClient, err := wireguardctrl.New()
	if err != nil {
		return nil, err
	}

	return &network{
		name:     name,
		wgClient: wgClient,
	}, nil
}

// Provider starts and configure wireguard network interface for providing service.
// It returns information required to establish connection to the service.
func (n *network) Provider() (Provider, error) {
	if _, err := n.wgClient.Device(n.name); err != nil {
		if err := exec.Command("ip", "link", "add", "dev", n.name, "type", "wireguard").Run(); err != nil {
			return Provider{}, err
		}
	}

	if err := exec.Command("ip", "address", "replace", "dev", n.name, "192.168.100.1/24").Run(); err != nil {
		return Provider{}, err
	}

	if err := exec.Command("ip", "link", "set", "dev", n.name, "up").Run(); err != nil {
		return Provider{}, err
	}

	// TODO wireguard provider listen port should be passed as startup argument
	port := 52820
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return Provider{}, err
	}

	if err := n.wgClient.ConfigureDevice(n.name, wgtypes.Config{
		PrivateKey:   &key,
		ListenPort:   &port,
		Peers:        nil,
		ReplacePeers: true,
	}); err != nil {
		return Provider{}, err
	}

	return Provider{
		PublicKey: key.PublicKey().String(),
		Endpoint:  fmt.Sprintf("%s:%d", n.publicIP, port),
	}, nil
}

// Consumer adds service consumer public key to the list of allowed peers.
// It returns information required to configure a consumer instance to allow connections.
func (n *network) Consumer() (Consumer, error) {
	peerKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return Consumer{}, err
	}

	// TODO constant peer IP should be replaced by some generation to allow more than one peer to connect
	_, peerIP, err := net.ParseCIDR("192.168.100.2/32")
	if err != nil {
		return Consumer{}, err
	}

	err = n.wgClient.ConfigureDevice(n.name, wgtypes.Config{
		Peers: []wgtypes.PeerConfig{{PublicKey: peerKey.PublicKey(), AllowedIPs: []net.IPNet{*peerIP}}}})
	if err != nil {
		return Consumer{}, err
	}

	return Consumer{
		PrivateKey: peerKey.String(),
		IP:         "192.168.100.2", // TODO Local IP should be calculated automatically for new connections.
	}, nil
}

// Close closes wireguard client and destroys wireguard network interface.
func (n *network) Close() error {
	if err := n.wgClient.Close(); err != nil {
		return err
	}

	return exec.Command("ip", "link", "del", "dev", n.name).Run()
}
