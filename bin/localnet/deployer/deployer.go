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

	"github.com/cheggaaa/pb"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mysteriumnetwork/payments/bindings"
)

func lookupBackend(rpcUrl string) (*ethclient.Client, chan bool, error) {
	ethClient, err := ethclient.Dial(rpcUrl)
	if err != nil {
		return nil, nil, err
	}

	block, err := ethClient.BlockByNumber(context.Background(), nil)
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("Latest known block is: ", block.NumberU64())

	progress, err := ethClient.SyncProgress(context.Background())
	if err != nil {
		return nil, nil, err
	}
	completed := make(chan bool)
	if progress != nil {
		fmt.Println("Client is in syncing state - any operations will be delayed until finished")
		go trackGethProgress(ethClient, progress, completed)
	} else {
		fmt.Println("Geth process fully synced")
		close(completed)
	}

	return ethClient, completed, nil
}

func trackGethProgress(client *ethclient.Client, lastProgress *ethereum.SyncProgress, completed chan<- bool) {
	bar := pb.New64(int64(lastProgress.HighestBlock)).
		SetTotal(int(lastProgress.CurrentBlock)).
		Start()
	defer bar.Finish()
	defer close(completed)
	for {
		progress, err := client.SyncProgress(context.Background())
		if err != nil {
			fmt.Println("Error querying client progress: " + err.Error())
			return
		}
		if progress == nil {
			bar.Finish()
			fmt.Println("Client in fully synced state. Proceeding...")
			return
		}
		bar.Set(int(progress.CurrentBlock))
		bar.SetTotal(int(progress.HighestBlock))
		time.Sleep(10 * time.Second)
	}
}

func main() {

	keyStoreDir := flag.String("keystore.directory", "", "Directory of keystore")
	etherAddress := flag.String("ether.address", "", "Account inside keystore to use for deployment")
	etherPassphrase := flag.String("ether.passphrase", "", "Passphrase for account unlocking")
	ethRPC := flag.String("geth.url", "", "RPC url of ethereum client")

	addr := common.HexToAddress(*etherAddress)

	flag.Parse()

	ks := keystore.NewKeyStore(*keyStoreDir, keystore.StandardScryptN, keystore.StandardScryptP)

	acc, err := ks.Find(accounts.Account{Address: addr})
	checkError("Find acc", err)

	err = ks.Unlock(acc, *etherPassphrase)
	checkError("Unlock acc", err)

	client, synced, err := lookupBackend(*ethRPC)
	checkError("lookup backend", err)

	chainID, err := client.NetworkID(context.Background())
	checkError("lookup chainid", err)

	transactor := &bind.TransactOpts{
		From: addr,
		Signer: func(signer types.Signer, address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			return ks.SignTx(acc, tx, chainID)
		},
		Context:  context.Background(),
		GasLimit: 1000000,
	}

	<-synced

	deployPaymentsv2Contracts(transactor, client)
}

func deployPaymentsv2Contracts(transactor *bind.TransactOpts, client *ethclient.Client) {
	transactor.Nonce = lookupLastNonce(transactor.From, client)
	mystTokenAddress, tx, _, err := bindings.DeployMystToken(transactor, client)
	checkError("Deploy token v2", err)
	checkTxStatus(client, tx)
	fmt.Println("v2 token address: ", mystTokenAddress.String())

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	mystDexAddress, tx, _, err := bindings.DeployMystDEX(transactor, client)
	checkError("Deploy mystDex v2", err)
	checkTxStatus(client, tx)
	fmt.Println("v2 mystDEX address: ", mystDexAddress.String())

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	channelImplAddress, tx, _, err := bindings.DeployChannelImplementation(transactor, client)
	checkError("Deploy channel impl v2", err)
	checkTxStatus(client, tx)
	fmt.Println("v2 channel impl address: ", channelImplAddress.String())

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	accountantImplAddress, tx, _, err := bindings.DeployAccountantImplementation(transactor, client)
	checkError("Deploy accountant impl v2", err)
	checkTxStatus(client, tx)
	fmt.Println("v2 accountant impl address: ", accountantImplAddress.String())

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	registrationFee := big.NewInt(0)
	minimalStake := big.NewInt(0)
	registryAddress, tx, _, err := bindings.DeployRegistry(
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
