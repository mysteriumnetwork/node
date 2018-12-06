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
	"errors"
	"net"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/nat"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/stretchr/testify/assert"
)

var (
	providerID = identity.FromAddress("provider-id")
)

var _ service.Service = NewManager(&fakeLocationResolver{}, &fakeIPResolver{}, &fakeConnectionEndpoint{})
var locationResolverStub = &fakeLocationResolver{
	err: nil,
	res: "LT",
}
var ipresolverStub = &fakeIPResolver{
	publicIPRes: "127.0.0.1",
	publicErr:   nil,
}

var connectionEndpointStub = &fakeConnectionEndpoint{}

func Test_Manager_Start(t *testing.T) {
	manager := newManagerStub(locationResolverStub, ipresolverStub)
	proposal, sessionConfigProvider, err := manager.Start(providerID)
	assert.NoError(t, err)
	assert.Exactly(
		t,
		dto_discovery.ServiceProposal{
			ServiceType: "wireguard",
			ServiceDefinition: wg.ServiceDefinition{
				Location: dto_discovery.Location{Country: "LT"},
			},
			PaymentMethodType: "WG",
			PaymentMethod: wg.Payment{
				Price: money.Money{
					Amount:   0,
					Currency: money.Currency("MYST"),
				},
			},
		},
		proposal,
	)
	sessionConfig, err := sessionConfigProvider()
	assert.NoError(t, err)
	assert.NotNil(t, sessionConfig)
}

func Test_Manager_Start_IPResolverErrs(t *testing.T) {
	fakeErr := errors.New("some error")
	ipResStub := &fakeIPResolver{
		publicIPRes: "127.0.0.1",
		publicErr:   fakeErr,
	}
	manager := newManagerStub(locationResolverStub, ipResStub)
	_, _, err := manager.Start(providerID)
	assert.Equal(t, fakeErr, err)
}

func Test_Manager_Start_LocResolverErrs(t *testing.T) {
	fakeErr := errors.New("some error")
	locResStub := &fakeLocationResolver{
		res: "LT",
		err: fakeErr,
	}
	manager := newManagerStub(locResStub, ipresolverStub)

	_, _, err := manager.Start(providerID)
	assert.Equal(t, fakeErr, err)
}

func Test_Manager_Wait(t *testing.T) {
	manager := newManagerStub(locationResolverStub, ipresolverStub)

	manager.Start(providerID)
	go func() {
		manager.Wait()
		assert.Fail(t, "Wait should be blocking")
	}()
	waitABit()
}

func Test_Manager_Stop(t *testing.T) {
	manager := newManagerStub(locationResolverStub, ipresolverStub)
	manager.Start(providerID)

	err := manager.Stop()
	assert.NoError(t, err)

	// Wait should not block after stopping
	manager.Wait()
}

// usually time.Sleep call gives a chance for other goroutines to kick in important when testing async code
func waitABit() {
	time.Sleep(10 * time.Millisecond)
}

type fakeLocationResolver struct {
	err error
	res string
}

// ResolveCountry performs a fake resolution
func (fr *fakeLocationResolver) ResolveCountry(ip string) (string, error) {
	return fr.res, fr.err
}

type fakeIPResolver struct {
	publicIPRes   string
	publicErr     error
	outboundIPRes string
	outboundErr   error
}

func (fir *fakeIPResolver) GetPublicIP() (string, error) {
	return fir.publicIPRes, fir.publicErr
}

func (fir *fakeIPResolver) GetOutboundIP() (string, error) {
	return fir.outboundIPRes, fir.outboundErr
}

type fakeConnectionEndpoint struct{}

func (fce *fakeConnectionEndpoint) Stop() error                            { return nil }
func (fce *fakeConnectionEndpoint) Start(_ *wg.ServiceConfig) error        { return nil }
func (fce *fakeConnectionEndpoint) Config() (wg.ServiceConfig, error)      { return wg.ServiceConfig{}, nil }
func (fce *fakeConnectionEndpoint) AddPeer(_ string, _ *net.UDPAddr) error { return nil }

func newManagerStub(locationResolver location.Resolver, ipResolver ip.Resolver) *Manager {
	return &Manager{
		locationResolver:   locationResolver,
		ipResolver:         ipResolver,
		connectionEndpoint: connectionEndpointStub,
		natService:         &serviceFake{},
	}
}

type serviceFake struct{}

func (service *serviceFake) Add(rule nat.RuleForwarding) {}
func (service *serviceFake) Start() error                { return nil }
func (service *serviceFake) Stop()                       {}
