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
	"fmt"
	"net"
	"os/exec"

	"github.com/mdlayher/wireguardctrl"
	"github.com/mdlayher/wireguardctrl/wgtypes"
)

// Provider is a configuration data required for establishing connection to the service provider.
type Provider struct {
	IP        net.IPNet
	PublicKey wgtypes.Key
	Endpoint  net.UDPAddr
}

// Consumer is a configuration data required to configure service consumer.
type Consumer struct {
	IP         net.IPNet
	PrivateKey wgtypes.Key // TODO peer private key should be generated on consumer side
}

type connectionEndpoint struct {
	iface    string
	endpoint net.UDPAddr
	wgClient *wireguardctrl.Client
}

// newConnectionEndpoint creates Wireguard client with predefined interface name and endpoint.
func newConnectionEndpoint(iface string, endpoint net.UDPAddr) (*connectionEndpoint, error) {
	wgClient, err := wireguardctrl.New()
	if err != nil {
		return nil, err
	}

	return &connectionEndpoint{
		iface:    iface,
		endpoint: endpoint,
		wgClient: wgClient,
	}, nil
}

// ProviderConfig starts and configure wireguard network interface for providing service.
// It returns information required to establish connection to the service.
func (n *connectionEndpoint) ProviderConfig(ip net.IPNet) (Provider, error) {
	if err := n.up(ip); err != nil {
		return Provider{}, err
	}

	// TODO wireguard provider listen port should be passed as startup argument
	port := 52820
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return Provider{}, err
	}

	if err := n.wgClient.ConfigureDevice(n.iface, wgtypes.Config{
		PrivateKey:   &key,
		ListenPort:   &port,
		ReplacePeers: true,
	}); err != nil {
		return Provider{}, err
	}

	return Provider{
		IP:        ip,
		PublicKey: key.PublicKey(),
		Endpoint:  n.endpoint,
	}, nil
}

// ConsumerConfig adds service consumer public key to the list of allowed peers.
// It returns information required to configure a consumer instance to allow connections.
func (n *connectionEndpoint) ConsumerConfig(ip net.IPNet) (Consumer, error) {
	peerKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return Consumer{}, err
	}

	err = n.wgClient.ConfigureDevice(n.iface, wgtypes.Config{
		Peers: []wgtypes.PeerConfig{{PublicKey: peerKey.PublicKey(), AllowedIPs: []net.IPNet{ip}}}})
	if err != nil {
		return Consumer{}, err
	}

	return Consumer{IP: ip, PrivateKey: peerKey}, nil
}

// Close closes wireguard client and destroys wireguard network interface.
func (n *connectionEndpoint) Close() error {
	if err := n.wgClient.Close(); err != nil {
		return err
	}

	return exec.Command("ip", "link", "del", "dev", n.iface).Run()
}

func (n *connectionEndpoint) up(net net.IPNet) error {
	if d, err := n.wgClient.Device(n.iface); err != nil || d.Name != n.iface {
		if out, err := exec.Command("ip", "link", "add", "dev", n.iface, "type", "wireguard").CombinedOutput(); err != nil {
			fmt.Println(string(out), err)
			return err
		}
	}

	if out, err := exec.Command("ip", "address", "replace", "dev", n.iface, net.String()).CombinedOutput(); err != nil {
		fmt.Println(string(out), err)
		return err
	}

	return exec.Command("ip", "link", "set", "dev", n.iface, "up").Run()
}
