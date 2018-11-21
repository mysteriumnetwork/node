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
	"sync"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/session"
)

const logPrefix = "[service-wireguard] "

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
}

// Config represent a Wireguard service provider configuration that will be passed to the consumer for establishing a connection
type Config struct {
	PublicKey string
	IP        string
	Endpoint  string
}

// Start starts service - does not block
func (manager *Manager) Start(providerID identity.Identity) (dto_discovery.ServiceProposal, session.ConfigProvider, error) {
	publicIP, err := manager.ipResolver.GetPublicIP()
	if err != nil {
		return dto_discovery.ServiceProposal{}, nil, err
	}

	sessionConfigProvider := func() (session.ServiceConfiguration, error) {
		return setupWireguard(publicIP)
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

	manager.process.Done()
	manager.isStarted = false
	log.Info(logPrefix, "Wireguard service stopped")
	return nil
}

func setupWireguard(publicIP string) (Config, error) {
	// TODO initialize wireguard interface.
	// out, err := exec.Command("ip", "link", "add", "dev", "wg0", "type", "wireguard").CombinedOutput()
	// out, err := exec.Command("ip", "address", "add", "dev", "wg0", "192.168.100.1/24").CombinedOutput()

	// TODO configure wireguard interface.
	// client, err := wireguardctrl.New()
	// if err != nil {
	// 	return Config{}, err
	// }

	// TODO wireguard device configuration like private key, listen port, peer list should survive through restarts.
	// TODO we need to have some persistent storage for it.
	// client.ConfigureDevice("wg0", wgtypes.Config{
	// 	PrivateKey: "8C6Pp0cI2tgLeYOMVbnMMLl/zN2blFK+OWGaIxX0bHY=",
	// 	ListenPort: 52820,
	// 	Peers:      nil,
	// })

	// TODO if the wireguard interface already configured we can get required parameters from it.
	// device, err := client.Device("wg0")
	// if err != nil {
	// 	return Config{}, err
	// }

	return Config{
		// TODO Local IP should be calculated automatically for new connections.
		IP:        "192.168.100.2",
		PublicKey: "rYx7j7p+xqBBPH+2lu19s2AzSzXzoedNLYGMBoOuDW0=", //device.PublicKey.String(),
		Endpoint:  "1.2.3.4:52820",                                //fmt.Sprintf("%s:%d", publicIP, device.ListenPort),
	}, nil
}
