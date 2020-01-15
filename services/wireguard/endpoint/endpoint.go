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
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/key"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
	"github.com/mysteriumnetwork/node/utils/netutil"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type wgClient interface {
	ConfigureDevice(name string, config wg.DeviceConfig, subnet net.IPNet) error
	ConfigureRoutes(iface string, ip net.IP) error
	DestroyDevice(name string) error
	AddPeer(iface string, peer wg.Peer) error
	RemovePeer(name string, publicKey string) error
	PeerStats() (*wg.Stats, error)
	Close() error
}

type connectionEndpoint struct {
	iface              string
	privateKey         string
	ipResolver         ip.Resolver
	ipAddr             net.IPNet
	endpoint           net.UDPAddr
	resourceAllocator  *resources.Allocator
	wgClient           wgClient
	releasePortMapping func()
	mapPort            func(port int) (releasePortMapping func())
	connectDelay       int // connect delay in milliseconds
}

// Start starts and configure wireguard network interface for providing service.
// If config is nil, required options will be generated automatically.
func (ce *connectionEndpoint) Start(config wg.StartConfig) error {
	if err := ce.cleanAbandonedInterfaces(); err != nil {
		return err
	}

	iface, err := ce.resourceAllocator.AllocateInterface()
	if err != nil {
		return errors.Wrap(err, "could not allocate interface")
	}

	var deviceConfig wg.DeviceConfig
	ce.iface = iface
	if config.Consumer == nil {
		pubIP, err := ce.ipResolver.GetPublicIP()
		if err != nil {
			return errors.Wrap(err, "could not get public IP")
		}
		port, err := ce.resourceAllocator.AllocatePort()
		if err != nil {
			return errors.Wrap(err, "could not allocate port")
		}
		privateKey, err := key.GeneratePrivateKey()
		if err != nil {
			return errors.Wrap(err, "could not generate private key")
		}
		ipAddr, err := ce.resourceAllocator.AllocateIPNet()
		if err != nil {
			return errors.Wrap(err, "could not allocate IP NET")
		}
		ce.ipAddr = ipAddr
		ce.ipAddr.IP = netutil.FirstIP(ce.ipAddr)
		ce.endpoint.IP = net.ParseIP(pubIP)
		ce.endpoint.Port = port
		ce.releasePortMapping = ce.mapPort(port)
		ce.privateKey = privateKey
		deviceConfig.ListenPort = ce.endpoint.Port
	} else {
		ce.ipAddr = config.Consumer.IPAddress
		ce.privateKey = config.Consumer.PrivateKey
		deviceConfig.ListenPort = config.Consumer.ListenPort
	}

	deviceConfig.PrivateKey = ce.privateKey
	if err := ce.wgClient.ConfigureDevice(ce.iface, deviceConfig, ce.ipAddr); err != nil {
		return errors.Wrap(err, "could not configure device")
	}
	return nil
}

// InterfaceName returns a connection endpoint interface name.
func (ce *connectionEndpoint) InterfaceName() string {
	return ce.iface
}

// AddPeer adds new wireguard peer to the wireguard device config.
func (ce *connectionEndpoint) AddPeer(iface string, peer wg.Peer) error {
	return ce.wgClient.AddPeer(iface, peer)
}

// RemovePeer removes a wireguard peer from the wireguard network interface.
func (ce *connectionEndpoint) RemovePeer(publicKey string) error {
	return ce.wgClient.RemovePeer(ce.iface, publicKey)
}

// PeerStats returns stats information about connected peer.
func (ce *connectionEndpoint) PeerStats() (*wg.Stats, error) {
	return ce.wgClient.PeerStats()
}

// Config provides wireguard service configuration for the current connection endpoint.
func (ce *connectionEndpoint) Config() (wg.ServiceConfig, error) {
	publicKey, err := key.PrivateKeyToPublicKey(ce.privateKey)
	if err != nil {
		return wg.ServiceConfig{}, err
	}

	pubIP, err := ce.ipResolver.GetPublicIP()
	if err != nil {
		return wg.ServiceConfig{}, err
	}
	outIP, err := ce.ipResolver.GetOutboundIPAsString()
	if err != nil {
		return wg.ServiceConfig{}, err
	}

	var config wg.ServiceConfig
	config.Provider.PublicKey = publicKey
	config.Provider.Endpoint = ce.endpoint
	config.Consumer.IPAddress = ce.ipAddr
	config.Consumer.IPAddress.IP = ce.consumerIP(ce.ipAddr)
	if outIP != pubIP {
		config.Consumer.ConnectDelay = ce.connectDelay
	}
	return config, nil
}

func (ce *connectionEndpoint) ConfigureRoutes(ip net.IP) error {
	return ce.wgClient.ConfigureRoutes(ce.iface, ip)
}

// Stop closes wireguard client and destroys wireguard network interface.
func (ce *connectionEndpoint) Stop() error {
	ce.releasePortMapping()

	if err := ce.wgClient.Close(); err != nil {
		return err
	}

	if err := ce.resourceAllocator.ReleaseIPNet(ce.ipAddr); err != nil {
		return err
	}

	return ce.resourceAllocator.ReleaseInterface(ce.iface)
}

func (ce *connectionEndpoint) cleanAbandonedInterfaces() error {
	ifaces, err := ce.resourceAllocator.AbandonedInterfaces()
	if err != nil {
		return err
	}

	for _, iface := range ifaces {
		if err := ce.wgClient.DestroyDevice(iface.Name); err != nil {
			log.Warn().Err(err).Msg("Failed to destroy abandoned interface: " + iface.Name)
		}
		log.Info().Msg("Abandoned interface destroyed: " + iface.Name)
	}

	return nil
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
