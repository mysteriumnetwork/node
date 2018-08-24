/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package e2e

import (
	"testing"

	"os"
	"time"

	paymentsRegistry "github.com/MysteriumNetwork/payments/registry"
	"github.com/cihub/seelog"
	"github.com/ethereum/go-ethereum/params"
	"github.com/mysterium/node/identity/registry"
	"github.com/mysterium/node/metadata"
	"github.com/stretchr/testify/assert"
)

func TestIdentityRegistrationFlow(t *testing.T) {

	tequilapi := newTequilaClient()

	mystIdentity, err := tequilapi.NewIdentity("")
	assert.NoError(t, err)
	seelog.Info("Created new identity: ", mystIdentity.Address)

	err = tequilapi.Unlock(mystIdentity.Address, "")
	assert.NoError(t, err)

	registrationStatus, err := tequilapi.RegistrationStatus(mystIdentity.Address)
	assert.NoError(t, err)
	assert.False(t, registrationStatus.Registered)

	err = registerIdentity(registrationStatus)
	assert.NoError(t, err)

	//now we check identity again
	newStatus, err := tequilapi.RegistrationStatus(mystIdentity.Address)
	assert.NoError(t, err)
	assert.True(t, newStatus.Registered)

}

func TestRegistrationNodeIdentity(t *testing.T) {
	// load e2e test node key
	userWallet, err := NewUserWallet("../bin/localnet/keystore")
	assert.NoError(t, err)

	//master account - owner of conctracts, and can issue tokens
	masterAccWallet, err := NewMainAccWallet("../bin/localnet/account")
	assert.NoError(t, err)

	//user gets some ethers from master acc
	err = masterAccWallet.GiveEther(userWallet.Owner, 1, params.Ether)
	assert.NoError(t, err)

	//user buys some tokens in exchange
	err = masterAccWallet.GiveTokens(userWallet.Owner, 3000)
	assert.NoError(t, err)

	//user gets some ethers from master acc
	err = masterAccWallet.GiveEther(userWallet.Owner, 1, params.Ether)
	assert.NoError(t, err)

	//user buys some tokens in exchange
	err = masterAccWallet.GiveTokens(userWallet.Owner, 1000)
	assert.NoError(t, err)

	//user allows payments to take some tokens
	err = userWallet.ApproveForPayments(1000)
	assert.NoError(t, err)

	identityHolder := paymentsRegistry.FromKeystore(userWallet.KS, userWallet.Owner)

	registrationData, err := paymentsRegistry.CreateRegistrationData(identityHolder)
	assert.NoError(t, err)

	// TODO: fix localnetdefinition for e2e test
	identityRegistry, err := registry.NewIdentityRegistry(userWallet.Backend, metadata.LocalnetDefinition.PaymentsContractAddress)

	registeredEventChan := make(chan int)
	stopLoop := make(chan int)
	go identityRegistry.WaitForRegistrationEvent(userWallet.Owner, registeredEventChan, stopLoop)

	go func() {
		//user registers identity
		err = userWallet.RegisterIdentityPlainData(registrationData)
		assert.NoError(t, err)

		registered, err := identityRegistry.IsRegistered(userWallet.Owner)
		assert.NoError(t, err)
		assert.True(t, registered)
	}()

	select {
	case <-registeredEventChan:
		break
	case <-time.After(10 * time.Second):
		close(stopLoop)
		t.Error("identity was not registered in time")
		t.Fail()
	}
}

func TestWaitForRegistrationEvent(t *testing.T) {
	defer os.RemoveAll("testdataoutput")
	userWallet, err := NewUserWallet("testdataoutput")
	assert.NoError(t, err)

	//master account - owner of conctracts, and can issue tokens
	masterAccWallet, err := NewMainAccWallet("../../bin/localnet/account")
	//masterAccWallet, err := NewMainAccWallet("../bin/localnet/account")
	assert.NoError(t, err)

	//user gets some ethers from master acc
	err = masterAccWallet.GiveEther(userWallet.Owner, 1, params.Ether)
	assert.NoError(t, err)

	//user buys some tokens in exchange
	err = masterAccWallet.GiveTokens(userWallet.Owner, 3000)
	assert.NoError(t, err)

	//user gets some ethers from master acc
	err = masterAccWallet.GiveEther(userWallet.Owner, 1, params.Ether)
	assert.NoError(t, err)

	//user buys some tokens in exchange
	err = masterAccWallet.GiveTokens(userWallet.Owner, 1000)
	assert.NoError(t, err)

	//user allows payments to take some tokens
	err = userWallet.ApproveForPayments(1000)
	assert.NoError(t, err)

	identityHolder := paymentsRegistry.FromKeystore(userWallet.KS, userWallet.Owner)

	registrationData, err := paymentsRegistry.CreateRegistrationData(identityHolder)
	assert.NoError(t, err)

	identityRegistry, err := registry.NewIdentityRegistry(userWallet.Backend, metadata.LocalnetDefinition.PaymentsContractAddress)

	registeredEventChan := make(chan int)
	stopLoop := make(chan int)
	go identityRegistry.WaitForRegistrationEvent(userWallet.Owner, registeredEventChan, stopLoop)

	go func() {
		//user registers identity
		err = userWallet.RegisterIdentityPlainData(registrationData)
		assert.NoError(t, err)

		registered, err := identityRegistry.IsRegistered(userWallet.Owner)
		assert.NoError(t, err)
		assert.True(t, registered)
	}()

	select {
	case <-registeredEventChan:
		break
	case <-time.After(10 * time.Second):
		close(stopLoop)
		t.Error("identity was not registered in time")
		t.Fail()
	}
}
