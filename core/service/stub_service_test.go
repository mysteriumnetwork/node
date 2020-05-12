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
	"sync"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/mysteriumnetwork/node/session"
)

var _ Service = &serviceFake{}

type serviceFake struct {
	mockProcess        chan struct{}
	onStartReturnError error
}

func (service *serviceFake) Serve(instance *Instance) error {
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

func (service *serviceFake) ProvideConfig(_ string, _ json.RawMessage, _ *net.UDPConn) (*session.ConfigParams, error) {
	return &session.ConfigParams{TraversalParams: traversal.Params{}}, nil
}

type mockDiscovery struct {
	wg sync.WaitGroup
}

func (mds *mockDiscovery) Start(ownIdentity identity.Identity, proposal market.ServiceProposal) {
	mds.wg.Add(1)
}
func (mds *mockDiscovery) Stop() {
	mds.wg.Done()
}

func (mds *mockDiscovery) Wait() {
	mds.wg.Wait()
}

// MockDiscoveryFactoryFunc returns a discovery factory which in turn returns the discovery service.
func MockDiscoveryFactoryFunc(ds Discovery) DiscoveryFactory {
	return func() Discovery {
		return ds
	}
}

// MockNATPinger returns a mock nat pinger, that really doesn't do much
type MockNATPinger struct{}

// Stop does nothing
func (mnp *MockNATPinger) Stop() {}
