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
	"github.com/mysteriumnetwork/payments/test"
	"github.com/rs/zerolog/log"
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

	err = deployPaymentsv2Contracts(transactor, client)
	checkError("deploy contracts", err)
}

func deployPaymentsv2Contracts(transactor *bind.TransactOpts, client *ethclient.Client) error {
	maxStake, ok := big.NewInt(0).SetString("62000000000000000000", 10)
	if !ok {
		return fmt.Errorf("failed to parse max stake")
	}
	stake, ok := big.NewInt(0).SetString("100000000000000000000", 10)
	if !ok {
		return fmt.Errorf("failed to parse stake")
	}

	addresses, err := test.DeployHermesWithDependencies(transactor, client, 10*time.Second, test.RegistryOpts{
		DexAddress:         common.HexToAddress("0x1"),
		MinimalHermesStake: big.NewInt(0),
	}, test.RegisterHermesOpts{
		Operator:        transactor.From,
		HermesStake:     stake,
		HermesFee:       400,
		MinChannelStake: big.NewInt(0),
		MaxChannelStake: maxStake,
		Url:             "http://hermes:8889",
	})
	if err != nil {
		return fmt.Errorf("failed to deploy hermes with dependencies: %w", err)
	}
	log.Debug().Interface("addresses", addresses.BaseContractAddresses).Msg("deployed base contracts")
	log.Debug().Interface("address", addresses.HermesAddress).Msg("deployed hermes contract")

	addresses2, err := test.DeployHermes(transactor, client, 10*time.Second, addresses.BaseContractAddresses, test.RegisterHermesOpts{
		Operator:        hermes2Address,
		HermesStake:     stake,
		HermesFee:       400,
		MinChannelStake: big.NewInt(0),
		MaxChannelStake: maxStake,
		Url:             "http://hermes2:8889",
	})
	if err != nil {
		return fmt.Errorf("failed to deploy hermes 2 with dependencies: %w", err)
	}
	log.Debug().Interface("address", addresses2.HermesAddress).Msg("deployed hermes 2 contract")

	transactorAddress := common.HexToAddress("0x3d2cdbab09d2c8d613556769f37b47c82a5e13bf")
	topupsAddress := common.HexToAddress("0xa29fb77b25181df094908b027821a7492ca4245b")
	value := big.NewInt(0).SetUint64(10000000000000000000)
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get chain id: %w", err)
	}

	for _, address := range []common.Address{transactorAddress, hermes2Address, topupsAddress} {
		err = test.TransferEth(transactor, client, chainID, address, value, 10*time.Second)
		if err != nil {
			return fmt.Errorf("failed to transfer eth: %w", err)
		}
	}

	for _, address := range []common.Address{transactorAddress, hermes2Address, addresses.HermesAddress, addresses2.HermesAddress, topupsAddress, transactor.From} {
		err = test.MintTokens(transactor, addresses.TokenV2Address, client, 10*time.Second, address, mystToMint)
		if err != nil {
			return fmt.Errorf("failed to mint myst: %w", err)
		}
	}

	log.Debug().Msg("sent funds to addresses")
	return nil
}

func checkError(context string, err error) {
	if err != nil {
		fmt.Println("Error at:", context, "value:", err.Error())
		os.Exit(1)
	}
}
