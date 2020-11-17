/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package registry

import (
	"math/big"
	"testing"

	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_ProviderRegistrar_StartsAndStops(t *testing.T) {
	mt := mockTransactor{}
	mrsp := mockRegistrationStatusProvider{}
	cfg := ProviderRegistrarConfig{}
	registrar := NewProviderRegistrar(&mt, &mrsp, cfg)

	done := make(chan struct{})

	go func() {
		err := registrar.start()
		assert.Nil(t, err)
		done <- struct{}{}
	}()
	registrar.stop()
	<-done
}

func Test_Provider_Registrar_needsHandling(t *testing.T) {
	mt := mockTransactor{}
	mrsp := mockRegistrationStatusProvider{}
	cfg := ProviderRegistrarConfig{}
	registrar := NewProviderRegistrar(&mt, &mrsp, cfg)

	mockEvent := queuedEvent{
		event:   servicestate.AppEventServiceStatus{},
		retries: 0,
	}

	assert.False(t, registrar.needsHandling(mockEvent))

	mockEvent.event.Status = "Running"
	assert.True(t, registrar.needsHandling(mockEvent))

	registrar.registeredIdentities["0x000"] = struct{}{}
	mockEvent.event.ProviderID = "0x000"
	assert.False(t, registrar.needsHandling(mockEvent))
}

func Test_Provider_Registrar_RegistersProvider(t *testing.T) {
	mt := mockTransactor{
		bountyResult: true,
	}
	mrsp := mockRegistrationStatusProvider{
		status: Unregistered,
	}
	cfg := ProviderRegistrarConfig{}
	registrar := NewProviderRegistrar(&mt, &mrsp, cfg)

	mockEvent := queuedEvent{
		event: servicestate.AppEventServiceStatus{
			Status:     "Running",
			ProviderID: "0xsuchIDManyWow",
		},
		retries: 0,
	}
	done := make(chan struct{})

	go func() {
		err := registrar.start()
		assert.Nil(t, err)
		done <- struct{}{}
	}()

	registrar.consumeServiceEvent(mockEvent.event)

	registrar.stop()
	<-done

	_, ok := registrar.registeredIdentities[mockEvent.event.ProviderID]
	assert.True(t, ok)
}

func Test_Provider_Registrar_Does_NotRegisterWithNoBounty(t *testing.T) {
	mt := mockTransactor{
		bountyResult: false,
	}
	mrsp := mockRegistrationStatusProvider{
		status: Unregistered,
	}
	cfg := ProviderRegistrarConfig{}
	registrar := NewProviderRegistrar(&mt, &mrsp, cfg)

	mockEvent := queuedEvent{
		event: servicestate.AppEventServiceStatus{
			Status:     "Running",
			ProviderID: "0xsuchIDManyWow",
		},
		retries: 0,
	}
	done := make(chan struct{})

	go func() {
		err := registrar.start()
		assert.Nil(t, err)
		done <- struct{}{}
	}()

	registrar.consumeServiceEvent(mockEvent.event)

	registrar.stop()
	<-done

	_, ok := registrar.registeredIdentities[mockEvent.event.ProviderID]
	assert.False(t, ok)
}

func Test_Provider_Registrar_Does_NotRegisterWithNoBounty_Testnet2(t *testing.T) {
	mt := mockTransactor{
		bountyResult: false,
	}
	mrsp := mockRegistrationStatusProvider{
		status: Unregistered,
	}
	cfg := ProviderRegistrarConfig{
		IsTestnet2: true,
	}
	registrar := NewProviderRegistrar(&mt, &mrsp, cfg)

	mockEvent := queuedEvent{
		event: servicestate.AppEventServiceStatus{
			Status:     "Running",
			ProviderID: "0xsuchIDManyWow",
		},
		retries: 0,
	}
	done := make(chan struct{})

	go func() {
		err := registrar.start()
		assert.Nil(t, err)
		done <- struct{}{}
	}()

	registrar.consumeServiceEvent(mockEvent.event)

	registrar.stop()
	<-done

	_, ok := registrar.registeredIdentities[mockEvent.event.ProviderID]
	assert.True(t, ok)
}

func Test_Provider_Registrar_FailsAfterRetries(t *testing.T) {
	mt := mockTransactor{}
	mrsp := mockRegistrationStatusProvider{
		err: errors.New("explosions everywhere"),
	}
	cfg := ProviderRegistrarConfig{
		MaxRetries: 5,
	}
	registrar := NewProviderRegistrar(&mt, &mrsp, cfg)

	mockEvent := queuedEvent{
		event: servicestate.AppEventServiceStatus{
			Status:     "Running",
			ProviderID: "0xsuchIDManyWow",
		},
		retries: 15,
	}
	done := make(chan struct{})

	go func() {
		err := registrar.start()
		assert.NotNil(t, err)
		done <- struct{}{}
	}()

	registrar.consumeServiceEvent(mockEvent.event)
	<-done
}

type mockRegistrationStatusProvider struct {
	status RegistrationStatus
	err    error
}

func (mrsp *mockRegistrationStatusProvider) GetRegistrationStatus(chainID int64, id identity.Identity) (RegistrationStatus, error) {
	return mrsp.status, mrsp.err
}

type mockTransactor struct {
	registerError error
	feesToReturn  FeesResponse
	feesError     error
	bountyError   error
	bountyResult  bool
}

func (mt *mockTransactor) FetchRegistrationFees() (FeesResponse, error) {
	return mt.feesToReturn, mt.feesError
}

func (mt *mockTransactor) RegisterIdentity(id string, stake, fee *big.Int, beneficiary string, chainID int64, referralToken *string) error {
	return mt.registerError
}

func (mt *mockTransactor) CheckIfRegistrationBountyEligible(identity identity.Identity) (bool, error) {
	return mt.bountyResult, mt.bountyError
}
