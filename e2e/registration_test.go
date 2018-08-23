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

	"time"

	"math/big"
	"os"

	"github.com/MysteriumNetwork/payments/cli/helpers"
	"github.com/MysteriumNetwork/payments/mysttoken/generated"
	"github.com/MysteriumNetwork/payments/registry"
	generatedRegistry "github.com/MysteriumNetwork/payments/registry/generated"
	"github.com/cihub/seelog"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/mysterium/node/blockchain"
	idRegistry "github.com/mysterium/node/identity/registry"
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

func _WaitForRegistrationEvent(t *testing.T) {

	tequilapi := newTequilaClient()

	mystIdentity, err := tequilapi.NewIdentity("")
	assert.NoError(t, err)
	seelog.Info("Created new identity: ", mystIdentity.Address)

	err = tequilapi.Unlock(mystIdentity.Address, "")
	assert.NoError(t, err)

	registrationStatus, err := tequilapi.RegistrationStatus(mystIdentity.Address)
	assert.NoError(t, err)
	assert.False(t, registrationStatus.Registered)

	ethClient, err := blockchain.NewClient(metadata.LocalnetDefinition.EtherClientRPC)
	assert.NoError(t, err)

	identityRegistry, err := idRegistry.NewIdentityRegistry(ethClient, metadata.LocalnetDefinition.PaymentsContractAddress)
	assert.NoError(t, err)

	registeredEventChan := make(chan bool)
	go identityRegistry.WaitForRegistrationEvent(common.HexToAddress(mystIdentity.Address), registeredEventChan)

	err = registerIdentity(registrationStatus)
	assert.NoError(t, err)

	select {
	case <-registeredEventChan:
		break
	case <-time.After(600 * time.Millisecond):
		t.Error("identity was not registered in time")
		t.Fail()
	}

	//now we check identity again
	newStatus, err := tequilapi.RegistrationStatus(mystIdentity.Address)
	assert.NoError(t, err)
	assert.True(t, newStatus.Registered)

}

func allocateFundsForIdentity(t *testing.T, identity common.Address) {
	//master account - owner of conctracts, and can issue tokens
	wd, err := os.Getwd()
	t.Log("Working dir", wd)

	masterAccWallet, err := NewMainAccWallet("../bin/localnet/account")
	t.Log(err)
	assert.NoError(t, err)

	//user gets some ethers from master acc
	err = masterAccWallet.GiveEther(identity, 1, params.Ether)
	assert.NoError(t, err)

	//user buys some tokens in exchange
	err = masterAccWallet.GiveTokens(identity, 1000)
	assert.NoError(t, err)
}

func TestWaitForRegistrationEvent(t *testing.T) {

	ethClient, err := blockchain.NewClient(metadata.E2EDefinition.EtherClientRPC)
	assert.NoError(t, err)

	erc20token, err := generated.NewMystTokenTransactor(metadata.E2EDefinition.PaymentsContractAddress, ethClient)
	assert.NoError(t, err)

	ks := keystore.NewKeyStore("testnet", keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.NewAccount("")
	assert.NoError(t, err)

	err = ks.Unlock(acc, "")
	assert.NoError(t, err)

	allocateFundsForIdentity(t, acc.Address)

	transactor := helpers.CreateNewKeystoreTransactor(ks, &acc)
	tx, err := erc20token.Approve(transactor, acc.Address, big.NewInt(3000))
	t.Log(tx)
	assert.NoError(t, err)

	identityRegistry, err := idRegistry.NewIdentityRegistry(ethClient, metadata.E2EDefinition.PaymentsContractAddress)

	registeredEventChan := make(chan bool)
	go identityRegistry.WaitForRegistrationEvent(acc.Address, registeredEventChan)

	identityHolder := registry.FromKeystore(ks, acc.Address)

	proofOfIdentity, err := registry.CreateRegistrationData(identityHolder)
	assert.NoError(t, err)

	paymentsContract, err := generatedRegistry.NewIdentityRegistryTransactor(metadata.E2EDefinition.PaymentsContractAddress, ethClient)

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
