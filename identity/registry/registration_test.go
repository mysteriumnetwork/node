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

package registry

import (
	"testing"

	"time"

	"math/big"

	"os"

	"github.com/MysteriumNetwork/payments/cli/helpers"
	"github.com/MysteriumNetwork/payments/mysttoken/generated"
	"github.com/MysteriumNetwork/payments/registry"
	generatedRegistry "github.com/MysteriumNetwork/payments/registry/generated"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/mysterium/node/blockchain"
	"github.com/mysterium/node/e2e"
	"github.com/mysterium/node/metadata"
	"github.com/stretchr/testify/assert"
)

func TestWaitForRegistrationEvent(t *testing.T) {
	defer os.RemoveAll("testdataoutput")
	userWallet, err := e2e.NewUserWallet("testdataoutput")
	assert.NoError(t, err)

	//master account - owner of conctracts, and can issue tokens
	//masterAccWallet, err := e2e.NewMainAccWallet("../../bin/localnet/account")
	masterAccWallet, err := e2e.NewMainAccWallet("../bin/localnet/account")
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

	identityHolder := registry.FromKeystore(userWallet.KS, userWallet.Owner)

	registrationData, err := registry.CreateRegistrationData(identityHolder)
	assert.NoError(t, err)

	identityRegistry, err := NewIdentityRegistry(userWallet.Backend, metadata.LocalnetDefinition.PaymentsContractAddress)

	registeredEventChan := make(chan bool)
	go identityRegistry.WaitForRegistrationEvent(userWallet.Owner, registeredEventChan)

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
		t.Error("identity was not registered in time")
		t.Fail()
	}
}

func _TestWaitForRegistrationEvent(t *testing.T) {
	ethClient, err := blockchain.NewClient(metadata.LocalnetDefinition.EtherClientRPC)
	assert.NoError(t, err)

	erc20token, err := generated.NewMystTokenTransactor(metadata.LocalnetDefinition.PaymentsContractAddress, ethClient)
	assert.NoError(t, err)

	ks := keystore.NewKeyStore("testnet", keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.NewAccount("")
	assert.NoError(t, err)

	err = ks.Unlock(acc, "")
	assert.NoError(t, err)

	// allocateFundsForIdentity(t, acc.Address)

	transactor := helpers.CreateNewKeystoreTransactor(ks, &acc)
	tx, err := erc20token.Approve(transactor, acc.Address, big.NewInt(3000))
	t.Log(tx)
	assert.NoError(t, err)

	identityRegistry, err := NewIdentityRegistry(ethClient, metadata.LocalnetDefinition.PaymentsContractAddress)

	registeredEventChan := make(chan bool)
	go identityRegistry.WaitForRegistrationEvent(acc.Address, registeredEventChan)

	identityHolder := registry.FromKeystore(ks, acc.Address)

	proofOfIdentity, err := registry.CreateRegistrationData(identityHolder)
	assert.NoError(t, err)

	paymentsContract, err := generatedRegistry.NewIdentityRegistryTransactor(metadata.LocalnetDefinition.PaymentsContractAddress, ethClient)

	tx, err = RegisterIdentity(proofOfIdentity, transactor, paymentsContract)
	t.Log(tx)
	assert.NoError(t, err)

	registered, err := identityRegistry.IsRegistered(acc.Address)
	assert.NoError(t, err)
	assert.True(t, registered)

	select {
	case <-registeredEventChan:
		break
	case <-time.After(600 * time.Millisecond):
		t.Error("identity was not registered in time")
		t.Fail()
	}
}

func RegisterIdentity(data *registry.RegistrationData, trOps *bind.TransactOpts, registryTransactor *generatedRegistry.IdentityRegistryTransactor) (*types.Transaction, error) {
	signature := data.Signature
	var pubKeyPart1 [32]byte
	var pubKeyPart2 [32]byte
	copy(pubKeyPart1[:], data.PublicKey.Part1)
	copy(pubKeyPart2[:], data.PublicKey.Part2)

	return registryTransactor.RegisterIdentity(trOps, pubKeyPart1, pubKeyPart2, signature.V, signature.R, signature.S)
}
