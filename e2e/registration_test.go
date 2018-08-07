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
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestIdentityRegistrationFlow(t *testing.T) {
	defer os.RemoveAll("testdataoutput")

	tequilapi := newTequilaClient()

	mystIdentity, err := tequilapi.NewIdentity("")
	assert.NoError(t, err)

	err = tequilapi.Unlock(mystIdentity.Address, "")
	assert.NoError(t, err)

	registrationStatus, err := tequilapi.RegistrationStatus(mystIdentity.Address)
	assert.NoError(t, err)
	assert.False(t, registrationStatus.Registered)

	//mystIdentity is not registered - register it, emulating interaction with wallet
	masterAccWallet, err := NewMainAccWallet("../bin/localnet/account")
	assert.NoError(t, err)

	userWallet, err := NewUserWallet("testdataoutput")
	assert.NoError(t, err)

	//user gets some ethers
	err = masterAccWallet.GiveEther(userWallet.Owner, 1, params.Ether)
	assert.NoError(t, err)

	//user buys some tokens
	err = masterAccWallet.GiveTokens(userWallet.Owner, 1000)
	assert.NoError(t, err)

	//user allows payments to take some tokens
	err = userWallet.ApproveForPayments(1000)
	assert.NoError(t, err)

	//user registers identity
	err = userWallet.RegisterIdentity(registrationStatus)
	assert.NoError(t, err)

	//now we check identity again
	newStatus, err := tequilapi.RegistrationStatus(mystIdentity.Address)
	assert.NoError(t, err)
	assert.True(t, newStatus.Registered)

}
