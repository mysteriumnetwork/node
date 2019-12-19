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
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/nat"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/stretchr/testify/assert"
)

var (
	providerID = identity.FromAddress("provider-id")
	pubIP      = "127.0.0.1"
	outIP      = "127.0.0.1"
	country    = "LT"
)

var connectionEndpointStub = &mockConnectionEndpoint{}

func Test_GetProposal(t *testing.T) {
	assert.Exactly(
		t,
		market.ServiceProposal{
			ServiceType: "wireguard",
			ServiceDefinition: wg.ServiceDefinition{
				Location:          market.Location{Country: country},
				LocationOriginate: market.Location{Country: country},
			},
			PaymentMethodType: "WG",
			PaymentMethod: wg.Payment{
				Price: money.Money{
					Amount:   1000000,
					Currency: money.Currency("MYST"),
				},
			},
		},
		GetProposal(location.Location{Country: country}),
	)
}

func Test_Manager_Serve(t *testing.T) {
	manager := newManagerStub(pubIP, outIP, country)

	go func() {
		err := manager.Serve(providerID)
		assert.NoError(t, err)
	}()

	sessionConfig, err := manager.ProvideConfig(json.RawMessage(`{"PublicKey": "gZfkZArbw9lqfl4Yzr1Kv3nqGlhe/ynH9KKRbzPFMGk="}`), nil)
	assert.NoError(t, err)
	assert.NotNil(t, sessionConfig)
}

func Test_Manager_Stop(t *testing.T) {
	manager := newManagerStub(pubIP, outIP, country)

	go func() {
		err := manager.Serve(providerID)
		assert.NoError(t, err)
	}()

	waitABit()
	err := manager.Stop()
	assert.NoError(t, err)
}

// usually time.Sleep call gives a chance for other goroutines to kick in important when testing async code
func waitABit() {
	time.Sleep(10 * time.Millisecond)
}

type mockConnectionEndpoint struct{}

func (mce *mockConnectionEndpoint) InterfaceName() string                        { return "mce0" }
func (mce *mockConnectionEndpoint) Stop() error                                  { return nil }
func (mce *mockConnectionEndpoint) Start(_ *wg.ServiceConfig) error              { return nil }
func (mce *mockConnectionEndpoint) Config() (wg.ServiceConfig, error)            { return wg.ServiceConfig{}, nil }
func (mce *mockConnectionEndpoint) AddPeer(_ string, _p wg.AddPeerOptions) error { return nil }
func (mce *mockConnectionEndpoint) RemovePeer(_ string) error                    { return nil }
func (mce *mockConnectionEndpoint) ConfigureRoutes(_ net.IP) error               { return nil }
func (mce *mockConnectionEndpoint) PeerStats() (*wg.Stats, error) {
	return &wg.Stats{LastHandshake: time.Now()}, nil
}

func newManagerStub(pub, out, country string) *Manager {
	return &Manager{
		ipResolver: ip.NewResolverMock("1.2.3.4"),
		natService: &serviceFake{},
		connectionEndpointFactory: func() (wg.ConnectionEndpoint, error) {
			return connectionEndpointStub, nil
		},
	}
}

type serviceFake struct{}

func (service *serviceFake) Setup(nat.Options) (rules []interface{}, err error) {
	return nil, nil
}
func (service *serviceFake) Del([]interface{}) error { return nil }
func (service *serviceFake) Enable() error           { return nil }
func (service *serviceFake) Disable() error          { return nil }
