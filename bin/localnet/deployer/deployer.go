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

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
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

	mystTokenAddress, tx, _, err := mysttoken.DeployMystToken(transactor, client)
	checkError("Deploy token", err)
	checkTxStatus(client, tx)
	fmt.Println("Token: ", mystTokenAddress.String())

	transactor.Nonce = big.NewInt(int64(tx.Nonce() + 1))
	paymentsAddress, tx, _, err := abigen.DeployIdentityPromises(transactor, client, mystTokenAddress, big.NewInt(100))
	checkError("Deploy payments", err)
	checkTxStatus(client, tx)

	fmt.Println("Payments: ", paymentsAddress.String())
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
