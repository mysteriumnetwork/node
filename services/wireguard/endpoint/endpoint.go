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
	"fmt"
	"net"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/key"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
	"github.com/mysteriumnetwork/node/utils/netutil"
)

// NewConnectionEndpoint returns new connection endpoint instance.
func NewConnectionEndpoint(resourceAllocator *resources.Allocator) (wg.ConnectionEndpoint, error) {
	wgClient, err := newWGClient()
	if err != nil {
		return nil, err
	}

	return &connectionEndpoint{
		wgClient:          wgClient,
		resourceAllocator: resourceAllocator,
	}, nil
}

type connectionEndpoint struct {
	cfg               wgcfg.DeviceConfig
	endpoint          net.UDPAddr
	resourceAllocator *resources.Allocator
	wgClient          WgClient
}

// StartConsumerMode starts and configure wireguard network interface running in consumer mode.
func (ce *connectionEndpoint) StartConsumerMode(cfg wgcfg.DeviceConfig) error {
	if err := ce.cleanAbandonedInterfaces(); err != nil {
		return err
	}
	iface, err := ce.resourceAllocator.AllocateInterface()
	if err != nil {
		return errors.Wrap(err, "could not allocate interface")
	}
	log.Debug().Msgf("Allocated interface: %s", iface)

	cfg.IfaceName = iface
	ce.cfg = cfg

	if err := ce.wgClient.ConfigureDevice(cfg); err != nil {
		return errors.Wrap(err, "could not configure device")
	}
	return nil
}

func (ce *connectionEndpoint) ReconfigureConsumerMode(cfg wgcfg.DeviceConfig) error {
	cfg.IfaceName = ce.cfg.IfaceName
	ce.cfg = cfg

	if err := ce.wgClient.ReConfigureDevice(cfg); err != nil {
		return fmt.Errorf("could not reconfigure device: %w", err)
	}

	return nil
}

func (ce *connectionEndpoint) StartProviderMode(publicIP string, config wgcfg.DeviceConfig) (err error) {
	if publicIP == "" {
		return errors.New("public IP is required")
	}
	if config.ListenPort == 0 {
		return errors.New("listen port is required")
	}

	if err := ce.cleanAbandonedInterfaces(); err != nil {
		return err
	}

	iface, err := ce.resourceAllocator.AllocateInterface()
	if err != nil {
		return errors.Wrap(err, "could not allocate interface")
	}

	config.IfaceName = iface
	config.Subnet.IP = netutil.FirstIP(config.Subnet)
	ce.cfg = config
	ce.endpoint = net.UDPAddr{IP: net.ParseIP(publicIP), Port: config.ListenPort}

	if err := ce.wgClient.ConfigureDevice(config); err != nil {
		return errors.Wrap(err, "could not configure device")
	}
	return nil
}

// InterfaceName returns a connection endpoint interface name.
func (ce *connectionEndpoint) InterfaceName() string {
	return ce.cfg.IfaceName
}

// PeerStats returns stats information about connected peer.
func (ce *connectionEndpoint) PeerStats() (*wgcfg.Stats, error) {
	return ce.wgClient.PeerStats(ce.cfg.IfaceName)
}

// Config provides wireguard service configuration for the current connection endpoint.
func (ce *connectionEndpoint) Config() (wg.ServiceConfig, error) {
	publicKey, err := key.PrivateKeyToPublicKey(ce.cfg.PrivateKey)
	if err != nil {
		return wg.ServiceConfig{}, err
	}

	var config wg.ServiceConfig
	config.Provider.PublicKey = publicKey
	config.Provider.Endpoint = ce.endpoint
	config.Consumer.IPAddress = ce.cfg.Subnet
	config.Consumer.IPAddress.IP = ce.consumerIP(ce.cfg.Subnet)
	return config, nil
}

// Stop closes wireguard client and destroys wireguard network interface.
func (ce *connectionEndpoint) Stop() error {
	if err := ce.wgClient.Close(); err != nil {
		return err
	}

	return ce.resourceAllocator.ReleaseInterface(ce.cfg.IfaceName)
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
