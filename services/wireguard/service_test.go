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
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
)

var (
	providerID = identity.FromAddress("provider-id")
)

var _ service.Service = NewManager(&fakeLocationResolver{}, &fakeIPResolver{})
var locationResolverStub = &fakeLocationResolver{
	err: nil,
	res: "LT",
}
var ipresolverStub = &fakeIPResolver{
	publicIPRes: "127.0.0.1",
	publicErr:   nil,
}

func Test_Manager_Stop(t *testing.T) {
	manager := NewManager(locationResolverStub, ipresolverStub)
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
