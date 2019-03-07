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
	"sync"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/session"
)

var _ Service = &serviceFake{}

type serviceFake struct {
	mockProcess        chan struct{}
	onStartReturnError error
}

func (service *serviceFake) Serve(identity.Identity) error {
	if service.mockProcess != nil {
		for range service.mockProcess {
		}
	}
	return service.onStartReturnError
}

func (service *serviceFake) Stop() error {
	if service.mockProcess != nil {
		close(service.mockProcess)
	}
	return nil
}

func (service *serviceFake) GetType() string {
	return "fake"
}

func (service *serviceFake) ProvideConfig(publicKey json.RawMessage) (session.ServiceConfiguration, session.DestroyCallback, error) {
	return struct{}{}, func() {}, nil
}

type mockDialogWaiter struct {
	contact  market.Contact
	stopErr  error
	serveErr error
	startErr error
}

func (mdw *mockDialogWaiter) Start() (market.Contact, error) {
	return mdw.contact, mdw.startErr
}

func (mdw *mockDialogWaiter) Stop() error {
	return mdw.stopErr
}

func (mdw *mockDialogWaiter) ServeDialogs(_ communication.DialogHandler) error {
	return mdw.serveErr
}

// MockDialogWaiterFactory returns a new instance of communication dialog waiter.
func MockDialogWaiterFactory(providerID identity.Identity, serviceType string) (communication.DialogWaiter, error) {
	return &mockDialogWaiter{}, nil
}

type mockDialogHandler struct {
}

func (mdh *mockDialogHandler) Handle(communication.Dialog) error {
	return nil
}

// MockDialogHandlerFactory creates a new mock dialog handler
func MockDialogHandlerFactory(market.ServiceProposal, session.ConfigNegotiator) communication.DialogHandler {
	return &mockDialogHandler{}
}

type mockDiscoveryService struct {
	wg sync.WaitGroup
}

func (mds *mockDiscoveryService) Start(ownIdentity identity.Identity, proposal market.ServiceProposal) {
	mds.wg.Add(1)
}
func (mds *mockDiscoveryService) Stop() {
	mds.wg.Done()
}

func (mds *mockDiscoveryService) Wait() {
	mds.wg.Wait()
}

// MockDiscoveryFactoryFunc returns a discovery factory which in turn returns the discovery service.
func MockDiscoveryFactoryFunc(ds DiscoveryService) DiscoveryFactory {
	return func() DiscoveryService {
		return ds
	}
}
