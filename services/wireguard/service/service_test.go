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

package service

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/policy/localcopy"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/dns"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
)

var (
	pubIP   = "127.0.0.1"
	outIP   = "127.0.0.1"
	country = "LT"
)

var connectionEndpointStub = &mockConnectionEndpoint{}

func Test_Manager_Stop(t *testing.T) {
	manager := newManagerStub(pubIP, outIP, country)

	service := service.NewInstance(
		identity.FromAddress("0x1"),
		"",
		nil,
		market.ServiceProposal{},
		servicestate.Running,
		nil,
		localcopy.NewRepository(),
		nil,
	)

	go func() {
		err := manager.Serve(service)
		assert.NoError(t, err)
	}()

	waitABit()
	err := manager.Stop()
	assert.NoError(t, err)
}

func Test_Manager_Stop_Datarace(t *testing.T) {
	manager := newManagerStub(pubIP, outIP, country)

	service := service.NewInstance(
		identity.FromAddress("0x1"),
		"",
		nil,
		market.ServiceProposal{},
		servicestate.Running,
		nil,
		localcopy.NewRepository(),
		nil,
	)
	manager.sessionCleanup = make(map[string]func())

	destroyClosure := func(sessionID string) func() {
		return func() {
			manager.sessionCleanupMu.Lock()
			defer manager.sessionCleanupMu.Unlock()

			_, ok := manager.sessionCleanup[sessionID]
			if !ok {
				return
			}
			delete(manager.sessionCleanup, sessionID)
		}
	}

	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		err := manager.Serve(service)
		assert.NoError(t, err)
	}()

	// normally the ProvideConfig() sets a callback, but in unit test we have to do it ourselves
	manager.sessionCleanupMu.Lock()
	manager.sessionCleanup["1"] = destroyClosure("1")
	manager.sessionCleanupMu.Unlock()

	waitABit()
	go func() {
		defer wg.Done()
		err := manager.Stop()
		assert.NoError(t, err)
	}()
	wg.Wait()
}

func Test_Manager_ProviderConfig_FailsWhenSessionConfigIsInvalid(t *testing.T) {
	manager := newManagerStub(pubIP, outIP, country)

	params, err := manager.ProvideConfig("", nil, nil)

	assert.Nil(t, params)
	assert.Error(t, err)
}

// usually time.Sleep call gives a chance for other goroutines to kick in important when testing async code
func waitABit() {
	time.Sleep(10 * time.Millisecond)
}

type mockConnectionEndpoint struct{}

func (mce *mockConnectionEndpoint) StartConsumerMode(config wgcfg.DeviceConfig) error { return nil }
func (mce *mockConnectionEndpoint) ReconfigureConsumerMode(config wgcfg.DeviceConfig) error {
	return nil
}

func (mce *mockConnectionEndpoint) StartProviderMode(ip string, config wgcfg.DeviceConfig) error {
	return nil
}
func (mce *mockConnectionEndpoint) InterfaceName() string                { return "mce0" }
func (mce *mockConnectionEndpoint) Stop() error                          { return nil }
func (mce *mockConnectionEndpoint) Config() (wg.ServiceConfig, error)    { return wg.ServiceConfig{}, nil }
func (mce *mockConnectionEndpoint) AddPeer(_ string, _ wgcfg.Peer) error { return nil }
func (mce *mockConnectionEndpoint) RemovePeer(_ string) error            { return nil }
func (mce *mockConnectionEndpoint) ConfigureRoutes(_ net.IP) error       { return nil }
func (mce *mockConnectionEndpoint) PeerStats() (wgcfg.Stats, error) {
	return wgcfg.Stats{LastHandshake: time.Now()}, nil
}
func (mce *mockConnectionEndpoint) Diag() error {
	return nil
}

func newManagerStub(pub, out, country string) *Manager {
	dnsHandler, _ := dns.ResolveViaSystem()

	return &Manager{
		done:       make(chan struct{}),
		ipResolver: ip.NewResolverMock("1.2.3.4"),
		natService: &serviceFake{},
		connEndpointFactory: func() (wg.ConnectionEndpoint, error) {
			return connectionEndpointStub, nil
		},
		dnsProxy: dns.NewProxy("", 0, dnsHandler),
	}
}

type serviceFake struct{}

func (service *serviceFake) Setup(nat.Options) (rules []interface{}, err error) {
	return nil, nil
}
func (service *serviceFake) Del([]interface{}) error { return nil }
func (service *serviceFake) Enable() error           { return nil }
func (service *serviceFake) Disable() error          { return nil }
