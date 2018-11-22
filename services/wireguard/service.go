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
	"errors"
	"fmt"
	"net"
	"os/exec"
	"sync"

	log "github.com/cihub/seelog"
	"github.com/mdlayher/wireguardctrl"
	"github.com/mdlayher/wireguardctrl/wgtypes"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/session"
)

const (
	logPrefix = "[service-wireguard] "

	interfaceName = "myst"
)

// ErrAlreadyStarted is the error we return when the start is called multiple times
var ErrAlreadyStarted = errors.New("Service already started")

// NewManager creates new instance of Wireguard service
func NewManager(locationResolver location.Resolver, ipResolver ip.Resolver) *Manager {
	return &Manager{
		locationResolver: locationResolver,
		ipResolver:       ipResolver,
	}
}

// Manager represents entrypoint for Wireguard service
type Manager struct {
	process          sync.WaitGroup
	locationResolver location.Resolver
	ipResolver       ip.Resolver
	isStarted        bool
	wgClient         *wireguardctrl.Client
}

// Config represent a Wireguard service provider configuration that will be passed to the consumer for establishing a connection
type Config struct {
	PublicKey string
	Endpoint  string
	PeerKey   string // TODO peer private key should be generated on consumer side
	PeerIP    string
}

// Start starts service - does not block
func (manager *Manager) Start(providerID identity.Identity) (dto_discovery.ServiceProposal, session.ConfigProvider, error) {
	publicIP, err := manager.ipResolver.GetPublicIP()
	if err != nil {
		return dto_discovery.ServiceProposal{}, nil, err
	}

	wgClient, err := wireguardctrl.New()
	if err != nil {
		return dto_discovery.ServiceProposal{}, nil, err
	}
	manager.wgClient = wgClient

	if err := manager.initInterface(interfaceName); err != nil {
		return dto_discovery.ServiceProposal{}, nil, err
	}

	sessionConfigProvider := func() (session.ServiceConfiguration, error) {
		return manager.peerConfig(interfaceName, publicIP)
	}

	if manager.isStarted {
		return dto_discovery.ServiceProposal{}, sessionConfigProvider, ErrAlreadyStarted
	}

	manager.process.Add(1)
	manager.isStarted = true
	log.Info(logPrefix, "Wireguard service started successfully")

	country, err := manager.locationResolver.ResolveCountry(publicIP)
	if err != nil {
		return dto_discovery.ServiceProposal{}, nil, err
	}

	proposal := dto_discovery.ServiceProposal{
		ServiceType: ServiceType,
		ServiceDefinition: ServiceDefinition{
			Location: dto_discovery.Location{Country: country},
		},
		PaymentMethodType: PaymentMethod,
		PaymentMethod: Payment{
			Price: money.NewMoney(0, money.CURRENCY_MYST),
		},
	}

	return proposal, sessionConfigProvider, nil
}

// Wait blocks until service is stopped
func (manager *Manager) Wait() error {
	if !manager.isStarted {
		return nil
	}
	manager.process.Wait()
	return nil
}

// Stop stops service
func (manager *Manager) Stop() error {
	if !manager.isStarted {
		return nil
	}

	if err := manager.wgClient.Close(); err != nil {
		return err
	}

	if err := manager.deleteInterface(interfaceName); err != nil {
		return err
	}

	manager.process.Done()
	manager.isStarted = false
	log.Info(logPrefix, "Wireguard service stopped")
	return nil
}

func (manager *Manager) initInterface(name string) error {
	if _, err := manager.wgClient.Device(name); err != nil {
		if err := exec.Command("ip", "link", "add", "dev", name, "type", "wireguard").Run(); err != nil {
			return err
		}
	}

	if err := exec.Command("ip", "address", "replace", "dev", name, "192.168.100.1/24").Run(); err != nil {
		return err
	}

	if err := exec.Command("ip", "link", "set", "dev", name, "up").Run(); err != nil {
		return err
	}

	// TODO wireguard provider listen port should be passed as startup argument
	port := 52820
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return err
	}

	return manager.wgClient.ConfigureDevice(name, wgtypes.Config{
		PrivateKey:   &key,
		ListenPort:   &port,
		Peers:        nil,
		ReplacePeers: true,
	})
}

func (manager *Manager) deleteInterface(name string) error {
	return exec.Command("ip", "link", "del", "dev", name).Run()
}

func (manager *Manager) peerConfig(name, publicIP string) (Config, error) {
	peerKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return Config{}, err
	}

	_, peerIP, err := net.ParseCIDR("192.168.100.2/32")
	if err != nil {
		return Config{}, err
	}

	err = manager.wgClient.ConfigureDevice(name, wgtypes.Config{
		Peers: []wgtypes.PeerConfig{wgtypes.PeerConfig{
			PublicKey:  peerKey.PublicKey(),
			AllowedIPs: []net.IPNet{*peerIP}}}})
	if err != nil {
		return Config{}, err
	}

	device, err := manager.wgClient.Device(name)
	if err != nil {
		return Config{}, err
	}

	return Config{
		// TODO Local IP should be calculated automatically for new connections.
		PublicKey: device.PublicKey.String(),
		Endpoint:  fmt.Sprintf("%s:%d", publicIP, device.ListenPort),
		PeerIP:    "192.168.100.2",
		PeerKey:   peerKey.String(),
	}, nil
}
