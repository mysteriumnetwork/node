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

package main

import (
	"context"
	"flag"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mysteriumnetwork/payment-bindings/generated"
	"github.com/mysteriumnetwork/payments/cli/helpers"
	"github.com/mysteriumnetwork/payments/contracts/abigen"
	"github.com/mysteriumnetwork/payments/mysttoken"
)

func main() {

	keyStoreDir := flag.String("keystore.directory", "", "Directory of keystore")
	etherAddress := flag.String("ether.address", "", "Account inside keystore to use for deployment")
	etherPassphrase := flag.String("ether.passphrase", "", "Passphrase for account unlocking")
	ethRPC := flag.String("geth.url", "", "RPC url of ethereum client")

	flag.Parse()

	ks := helpers.GetKeystore(*keyStoreDir)

	acc, err := helpers.GetUnlockedAcc(*etherAddress, *etherPassphrase, ks)
	checkError("Unlock acc", err)

	transactor := helpers.CreateNewKeystoreTransactor(ks, acc)

	client, synced, err := helpers.LookupBackend(*ethRPC)
	checkError("backend lookup", err)
	<-synced

	// we still need to deploy legacy contracts to blockchain to be backwards compatible with current usage
	deployLegacyContracts(transactor, client)

	deployPaymentsv2Contracts(transactor, client)
}

func deployPaymentsv2Contracts(transactor *bind.TransactOpts, client *ethclient.Client) {
	transactor.Nonce = lookupLastNonce(transactor.From, client)
	mystTokenAddress, tx, _, err := generated.DeployMystToken(transactor, client)
	checkError("Deploy token v2", err)
	checkTxStatus(client, tx)
	fmt.Println("v2 token address: ", mystTokenAddress.String())

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	mystDexAddress, tx, _, err := generated.DeployMystDEX(transactor, client)
	checkError("Deploy mystDex v2", err)
	checkTxStatus(client, tx)
	fmt.Println("v2 mystDEX address: ", mystDexAddress.String())

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	channelImplAddress, tx, _, err := generated.DeployChannelImplementation(transactor, client)
	checkError("Deploy channel impl v2", err)
	checkTxStatus(client, tx)
	fmt.Println("v2 channel impl address: ", channelImplAddress.String())

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	accountantImplAddress, tx, _, err := generated.DeployAccountantImplementation(transactor, client)
	checkError("Deploy accountant impl v2", err)
	checkTxStatus(client, tx)
	fmt.Println("v2 accountant impl address: ", accountantImplAddress.String())

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	registrationFee := big.NewInt(0)
	minimalStake := big.NewInt(0)
	registryAddress, tx, _, err := generated.DeployRegistry(
		transactor,
		client,
		mystTokenAddress,
		mystDexAddress,
		channelImplAddress,
		accountantImplAddress,
		registrationFee,
		minimalStake,
	)
	checkError("Deploy registry v2", err)
	checkTxStatus(client, tx)
	fmt.Println("v2 registry address: ", registryAddress.String())
}

func deployLegacyContracts(transactor *bind.TransactOpts, client *ethclient.Client) uint64 {
	mystTokenAddress, tx, _, err := mysttoken.DeployMystToken(transactor, client)
	checkError("Deploy token", err)
	checkTxStatus(client, tx)
	fmt.Println("(deprecated) Token: ", mystTokenAddress.String())

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	paymentsAddress, tx, _, err := abigen.DeployIdentityPromises(transactor, client, mystTokenAddress, big.NewInt(100))
	checkError("Deploy payments", err)
	checkTxStatus(client, tx)
	fmt.Println("(deprecated) Payments: ", paymentsAddress.String())
	return tx.Nonce()
}

func checkError(context string, err error) {
	if err != nil {
		fmt.Println("Error at:", context, "value:", err.Error())
		os.Exit(1)
	}
}

func checkTxStatus(client *ethclient.Client, tx *types.Transaction) {

	//wait for transaction to be mined at most 10 seconds
	for i := 0; i < 10; i++ {
		_, pending, err := client.TransactionByHash(context.Background(), tx.Hash())
		checkError("Get tx by hash", err)
		if pending {
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}

	receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
	checkError("Fetch tx receipt", err)
	if receipt.Status != 1 {
		fmt.Println("Receipt status expected to be 1")
		os.Exit(1)
	}
}

func lookupLastNonce(addr common.Address, client *ethclient.Client) *big.Int {
	nonce, err := client.NonceAt(context.Background(), addr, nil)
	checkError("Lookup last nonce", err)
	return big.NewInt(int64(nonce))
}
