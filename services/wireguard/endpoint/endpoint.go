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

	"github.com/mysteriumnetwork/node/core/location"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/key"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
	"github.com/mysteriumnetwork/node/utils/netutil"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// NewConnectionEndpoint returns new connection endpoint instance.
func NewConnectionEndpoint(
	location *location.ServiceLocationInfo,
	resourceAllocator *resources.Allocator,
	connectDelay int) (wg.ConnectionEndpoint, error) {

	wgClient, err := newWGClient()
	if err != nil {
		return nil, err
	}

	return &connectionEndpoint{
		location:          location,
		wgClient:          wgClient,
		resourceAllocator: resourceAllocator,
		connectDelay:      connectDelay,
	}, nil
}

type connectionEndpoint struct {
	location          *location.ServiceLocationInfo
	iface             string
	privateKey        string
	ipAddr            net.IPNet
	endpoint          net.UDPAddr
	resourceAllocator *resources.Allocator
	wgClient          wgClient
	connectDelay      int // connect delay in milliseconds
}

// StartConsumerMode starts and configure wireguard network interface running in consumer mode.
func (ce *connectionEndpoint) StartConsumerMode(config wg.ConsumerModeConfig) error {
	if err := ce.cleanAbandonedInterfaces(); err != nil {
		return err
	}

	iface, err := ce.resourceAllocator.AllocateInterface()
	if err != nil {
		return errors.Wrap(err, "could not allocate interface")
	}

	ce.iface = iface
	ce.ipAddr = config.IPAddress
	ce.privateKey = config.PrivateKey

	deviceConfig := wg.DeviceConfig{
		IfaceName:  ce.iface,
		Subnet:     ce.ipAddr,
		ListenPort: config.ListenPort,
		PrivateKey: ce.privateKey,
	}
	if err := ce.wgClient.ConfigureDevice(deviceConfig); err != nil {
		return errors.Wrap(err, "could not configure device")
	}
	return nil
}

func (ce *connectionEndpoint) StartProviderMode(config wg.ProviderModeConfig) (err error) {
	if err := ce.cleanAbandonedInterfaces(); err != nil {
		return err
	}

	ce.iface, err = ce.resourceAllocator.AllocateInterface()
	if err != nil {
		return errors.Wrap(err, "could not allocate interface")
	}

	ce.privateKey, err = key.GeneratePrivateKey()
	if err != nil {
		return errors.Wrap(err, "could not generate private key")
	}
	ce.ipAddr, err = ce.resourceAllocator.AllocateIPNet()
	if err != nil {
		return errors.Wrap(err, "could not allocate IP NET")
	}
	ce.ipAddr.IP = netutil.FirstIP(ce.ipAddr)
	ce.endpoint = net.UDPAddr{IP: net.ParseIP(ce.location.PubIP), Port: config.ListenPort}

	deviceConfig := wg.DeviceConfig{
		IfaceName:  ce.iface,
		Subnet:     ce.ipAddr,
		ListenPort: ce.endpoint.Port,
		PrivateKey: ce.privateKey,
	}
	if err := ce.wgClient.ConfigureDevice(deviceConfig); err != nil {
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

	var config wg.ServiceConfig
	config.Provider.PublicKey = publicKey
	config.Provider.Endpoint = ce.endpoint
	config.Consumer.IPAddress = ce.ipAddr
	config.Consumer.IPAddress.IP = ce.consumerIP(ce.ipAddr)
	if ce.location.BehindNAT() {
		config.Consumer.ConnectDelay = ce.connectDelay
	}
	return config, nil
}

func (ce *connectionEndpoint) ConfigureRoutes(ip net.IP) error {
	return ce.wgClient.ConfigureRoutes(ce.iface, ip)
}

// Stop closes wireguard client and destroys wireguard network interface.
func (ce *connectionEndpoint) Stop() error {
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
