/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mysteriumnetwork/payments/bindings"
)

var hermes2Address = common.HexToAddress("0x761f2bb3e7ad6385a4c7833c5a26a8ddfdabf9f3")
var mystToMint, _ = big.NewInt(0).SetString("1250000000000000000000", 10)

func main() {
	keyStoreDir := flag.String("keystore.directory", "", "Directory of keystore")
	etherAddress := flag.String("ether.address", "", "Account inside keystore to use for deployment")
	etherPassword := flag.String("ether.passphrase", "", "key of the account")
	ethRPC := flag.String("geth.url", "", "RPC url of ethereum client")
	flag.Parse()

	addr := common.HexToAddress(*etherAddress)
	ks := keystore.NewKeyStore(*keyStoreDir, keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.Find(accounts.Account{Address: addr})
	checkError("find account", err)

	err = ks.Unlock(acc, *etherPassword)
	checkError("unlock account", err)

	client, err := ethclient.Dial(*ethRPC)
	checkError("lookup backend", err)

	chainID, err := client.NetworkID(context.Background())
	checkError("lookup chainid", err)

	transactor := &bind.TransactOpts{
		From: addr,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			return ks.SignTx(acc, tx, chainID)
		},
		Context:  context.Background(),
		GasLimit: 6721975,
	}

	deployPaymentsv2Contracts(transactor, client, ks)
}

func deployPaymentsv2Contracts(transactor *bind.TransactOpts, client *ethclient.Client, ks *keystore.KeyStore) {
	time.Sleep(time.Second * 3)

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	gasPrice, err := client.SuggestGasPrice(context.Background())
	checkError("get gasPrice", err)
	gasLimit := uint64(21000) // in units
	checkError("lookup chainid", err)

	oldToken, tx, _, err := bindings.DeployOldMystToken(transactor, client)
	checkError("Deploy old token v2", err)
	checkTxStatus(client, tx)

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	mystTokenAddress, tx, _, err := bindings.DeployMystToken(transactor, client, oldToken)
	checkError("Deploy token v2", err)
	checkTxStatus(client, tx)
	fmt.Println("v2 token address: ", mystTokenAddress.String())

	transactor.Nonce = lookupLastNonce(transactor.From, client)

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	channelImplAddress, tx, _, err := bindings.DeployChannelImplementation(transactor, client)
	checkError("Deploy channel impl v2", err)
	checkTxStatus(client, tx)
	fmt.Println("v2 channel impl address: ", channelImplAddress.String())

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	hermesImplAddress, tx, _, err := bindings.DeployHermesImplementation(transactor, client)
	checkError("Deploy hermes impl v2", err)
	checkTxStatus(client, tx)
	fmt.Println("v2 hermes impl address: ", hermesImplAddress.String())

	transactor.Nonce = lookupLastNonce(transactor.From, client)

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	minimalStake := big.NewInt(0)
	registryAddress, tx, _, err := bindings.DeployRegistry(
		transactor,
		client,
	)
	checkError("Deploy registry v2", err)
	checkTxStatus(client, tx)
	fmt.Println("v2 registry address: ", registryAddress.String())

	txer, err := bindings.NewRegistryTransactor(registryAddress, client)
	checkError("create registry transactor", err)

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	// mystDexAddress dex is not being used so no point in deploying it.
	mystDexAddress := common.HexToAddress("0x0000123123123123")
	_, err = txer.Initialize(transactor, mystTokenAddress, mystDexAddress, minimalStake, channelImplAddress, hermesImplAddress, common.Address{})
	checkError("initialize registry", err)

	ts, err := bindings.NewMystTokenTransactor(mystTokenAddress, client)
	checkError("myst transactor", err)

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	tx, err = ts.Mint(transactor, transactor.From, mystToMint)
	checkError("mint myst", err)
	checkTxStatus(client, tx)

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	tx, err = ts.Approve(transactor, registryAddress, mystToMint)
	checkError("allow myst", err)
	checkTxStatus(client, tx)

	rt, err := bindings.NewRegistryTransactor(registryAddress, client)
	checkError("registry transactor", err)

	maxStake, _ := big.NewInt(0).SetString("62000000000000000000", 10)
	stake, _ := big.NewInt(0).SetString("100000000000000000000", 10)
	transactor.Nonce = lookupLastNonce(transactor.From, client)
	tx, err = rt.RegisterHermes(
		transactor,
		transactor.From,
		stake,
		400,
		big.NewInt(1000),
		maxStake,
		[]byte("http://hermes:8889"),
	)
	checkError("register hermes", err)
	checkTxStatus(client, tx)

	rc, err := bindings.NewRegistryCaller(registryAddress, client)
	checkError("registry caller", err)

	accs, err := rc.GetHermesAddress(&bind.CallOpts{
		Context: context.Background(),
		From:    transactor.From,
	}, transactor.From, big.NewInt(0))
	checkError("get hermes address", err)
	fmt.Println("registered hermes", accs.Hex())

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	tx, err = ts.Mint(transactor, accs, mystToMint)
	checkError("mint myst for hermes", err)
	checkTxStatus(client, tx)

	// transfer eth to hermes2operator
	value := big.NewInt(0).SetUint64(10000000000000000000)
	gasLimit = uint64(21000)
	gasPrice, err = client.SuggestGasPrice(context.Background())
	checkError("suggest gas price", err)

	transactor.Nonce = lookupLastNonce(transactor.From, client)

	var data []byte
	tx = types.NewTransaction(transactor.Nonce.Uint64(), hermes2Address, value, gasLimit, gasPrice, data)

	signedTx, err := transactor.Signer(transactor.From, tx)
	checkError("sign tx", err)

	err = client.SendTransaction(context.Background(), signedTx)
	checkError("transfer eth", err)
	checkTxStatus(client, signedTx)

	// mint myst to hermes2operator
	transactor.Nonce = lookupLastNonce(transactor.From, client)
	tx, err = ts.Mint(transactor, hermes2Address, mystToMint)
	checkError("mint myst for hermes2", err)
	checkTxStatus(client, tx)

	// register hermes2
	accs = registerHermes2(ks, client, registryAddress, mystTokenAddress)
	transactor.Nonce = lookupLastNonce(transactor.From, client)

	// mint myst to registered hermes 2
	tx, err = ts.Mint(transactor, accs, mystToMint)
	checkError("mint myst for hermes2", err)
	checkTxStatus(client, tx)
	transactor.Nonce = lookupLastNonce(transactor.From, client)

	// transfer eth and myst for transactions during the e2e tests
	value = big.NewInt(0).SetUint64(10000000000000000000)
	gasLimit = uint64(21000)
	gasPrice, err = client.SuggestGasPrice(context.Background())
	checkError("suggest gas price", err)
	transactor.Nonce = lookupLastNonce(transactor.From, client)

	tx = types.NewTransaction(transactor.Nonce.Uint64(), common.HexToAddress("0xa29fb77b25181df094908b027821a7492ca4245b"), value, gasLimit, gasPrice, data)

	signedTx, err = transactor.Signer(transactor.From, tx)
	checkError("sign tx", err)

	err = client.SendTransaction(context.Background(), signedTx)
	checkError("transfer eth", err)
	checkTxStatus(client, signedTx)
	transactor.Nonce = lookupLastNonce(transactor.From, client)

	tx, err = ts.Mint(transactor, common.HexToAddress("0xa29fb77b25181df094908b027821a7492ca4245b"), mystToMint)
	checkError("mint myst for future top-ups during tests", err)
	checkTxStatus(client, tx)
	transactor.Nonce = lookupLastNonce(transactor.From, client)

	// print some state from hermes2
	caller, err := bindings.NewHermesImplementationCaller(accs, client)
	stake, err = caller.AvailableBalance(&bind.CallOpts{})
	fmt.Println("balance available hermes", stake.Uint64())

	fee, err := caller.CalculateHermesFee(&bind.CallOpts{}, big.NewInt(15931))
	fmt.Println("fee hermes", fee.Uint64())

	balance, _ := caller.MinimalExpectedBalance(&bind.CallOpts{})
	fmt.Println(" minimal balance  hermes", balance.Uint64())

	status, _ := caller.GetStatus(&bind.CallOpts{})
	fmt.Println(" status   hermes", status)

	min, max, _ := caller.GetStakeThresholds(&bind.CallOpts{})
	fmt.Println(" min max hermes ", min.Uint64(), " ", max.Uint64())

	tc, _ := bindings.NewMystTokenCaller(mystTokenAddress, client)
	tokenBalance, _ := tc.BalanceOf(&bind.CallOpts{}, accs)
	fmt.Println(" tokenBalance ", tokenBalance.Uint64())
}

func registerHermes2(ks *keystore.KeyStore, client *ethclient.Client, registryAddress, mystTokenAddress common.Address) common.Address {
	acc, err := ks.Find(accounts.Account{Address: hermes2Address})
	checkError("find account", err)

	err = ks.Unlock(acc, "")
	checkError("unlock account", err)

	chainID, err := client.NetworkID(context.Background())
	checkError("lookup chainid", err)

	transactor := &bind.TransactOpts{
		From: hermes2Address,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			return ks.SignTx(acc, tx, chainID)
		},
		Context:  context.Background(),
		GasLimit: 6721975,
	}

	ts, err := bindings.NewMystTokenTransactor(mystTokenAddress, client)
	checkError("myst transactor", err)

	// allow myst to registry
	transactor.Nonce = lookupLastNonce(transactor.From, client)
	tx, err := ts.Approve(transactor, registryAddress, mystToMint)
	checkError("allow myst", err)
	checkTxStatus(client, tx)

	rt, err := bindings.NewRegistryTransactor(registryAddress, client)
	checkError("registry transactor", err)

	minStake, _ := big.NewInt(0).SetString("1000000000000000000", 10)
	maxStake, _ := big.NewInt(0).SetString("62000000000000000000", 10)
	stake, _ := big.NewInt(0).SetString("100000000000000000000", 10)

	transactor.Nonce = lookupLastNonce(transactor.From, client)
	// register hermes
	tx, err = rt.RegisterHermes(
		transactor,
		transactor.From,
		stake,
		400,
		minStake,
		maxStake,
		[]byte("http://hermes2:8889"),
	)
	checkError("register hermes 2", err)
	checkTxStatus(client, tx)

	rc, err := bindings.NewRegistryCaller(registryAddress, client)
	checkError("registry caller", err)

	// get hermes address
	accs, err := rc.GetHermesAddress(&bind.CallOpts{
		Context: context.Background(),
		From:    transactor.From,
	}, transactor.From, big.NewInt(0))
	checkError("get hermes2 address", err)
	fmt.Println("registered hermes2", accs.Hex())

	return accs
}

func checkError(context string, err error) {
	if err != nil {
		fmt.Println("Error at:", context, "value:", err.Error())
		os.Exit(1)
	}
}
func lookupLastNonce(addr common.Address, client *ethclient.Client) *big.Int {
	nonce, err := client.NonceAt(context.Background(), addr, nil)
	checkError("Lookup last nonce", err)
	return big.NewInt(int64(nonce))
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
