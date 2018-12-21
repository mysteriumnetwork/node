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
	"context"
	"errors"
	"flag"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
	"github.com/mysteriumnetwork/payments/cli/helpers"
	"github.com/mysteriumnetwork/payments/contracts/abigen"
	"github.com/mysteriumnetwork/payments/mysttoken"
)

var (
	// addresses should match those deployed in e2e test environment
	tokenAddress    = common.HexToAddress("0x0222eb28e1651E2A8bAF691179eCfB072457f00c")
	paymentsAddress = common.HexToAddress("0x1955141ba8e77a5B56efBa8522034352c94f77Ea")

	// deployer of contracts and main acc with ethereum
	deployerKeystoreDir = flag.String("deployer.keystore-directory", "", "Directory of deployer's keystore")
	deployerAddress     = flag.String("deployer.address", "", "Deployer's account inside keystore")
	deployerPassphrase  = flag.String("deployer.passphrase", "", "Deployer's passphrase for account unlocking")
)

// CliWallet represents operations which can be done with user controlled account
type CliWallet struct {
	txOpts           *bind.TransactOpts
	Owner            common.Address
	backend          *ethclient.Client
	identityRegistry abigen.IdentityPromisesTransactorSession
	tokens           mysttoken.MystTokenTransactorSession
	ks               *keystore.KeyStore
}

// RegisterIdentity registers identity with given data on behalf of user
func (wallet *CliWallet) RegisterIdentity(dto tequilapi_client.RegistrationDataDTO) error {
	var Pub1 [32]byte
	var Pub2 [32]byte
	var S [32]byte
	var R [32]byte

	copy(Pub1[:], common.FromHex(dto.PublicKey.Part1))
	copy(Pub2[:], common.FromHex(dto.PublicKey.Part2))
	copy(R[:], common.FromHex(dto.Signature.R))
	copy(S[:], common.FromHex(dto.Signature.S))

	tx, err := wallet.identityRegistry.RegisterIdentity(Pub1, Pub2, dto.Signature.V, R, S)
	if err != nil {
		return err
	}
	return wallet.checkTxResult(tx)
}

// GiveEther transfers ether to given address
func (wallet *CliWallet) GiveEther(address common.Address, amount, units int64) error {

	amountInWei := new(big.Int).Mul(big.NewInt(amount), big.NewInt(units))

	nonce, err := wallet.backend.PendingNonceAt(context.Background(), wallet.Owner)
	if err != nil {
		return err
	}
	gasPrice, err := wallet.backend.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}

	tx := types.NewTransaction(nonce, address, amountInWei, params.TxGas, gasPrice, nil)

	signedTx, err := wallet.txOpts.Signer(types.HomesteadSigner{}, wallet.Owner, tx)
	if err != nil {
		return err
	}

	err = wallet.backend.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}
	return wallet.checkTxResult(signedTx)
}

// GiveTokens gives myst tokens to specified address
func (wallet *CliWallet) GiveTokens(address common.Address, amount int64) error {
	tx, err := wallet.tokens.Mint(address, big.NewInt(amount))
	if err != nil {
		return err
	}
	return wallet.checkTxResult(tx)
}

// ApproveForPayments allows specified amount of ERC20 tokens to be spend by payments contract
func (wallet *CliWallet) ApproveForPayments(amount int64) error {
	tx, err := wallet.tokens.Approve(paymentsAddress, big.NewInt(amount))
	if err != nil {
		return err
	}
	return wallet.checkTxResult(tx)
}

func (wallet *CliWallet) checkTxResult(tx *types.Transaction) error {
	for i := 0; i < 10; i++ {
		_, pending, err := wallet.backend.TransactionByHash(context.Background(), tx.Hash())
		switch {
		case err != nil:
			return err
		case pending:
			time.Sleep(1 * time.Second)
		case !pending:
			break
		}
	}

	receipt, err := wallet.backend.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		return err
	}
	if receipt.Status != 1 {
		return errors.New("tx marked as failed")
	}
	return nil
}

// NewDeployerWallet initializes wallet with main localnet account private key (owner of ERC20, payments and lots of ether)
func NewDeployerWallet() (*CliWallet, error) {
	ks := initKeyStore(*deployerKeystoreDir)
	return newCliWallet(common.HexToAddress(*deployerAddress), *deployerPassphrase, ks)
}

// NewUserWallet initializes wallet with generated account with specified keystore
func NewUserWallet(keystoreDir string) (*CliWallet, error) {
	ks := initKeyStore(keystoreDir)
	acc, err := ks.NewAccount("")
	if err != nil {
		return nil, err
	}
	return newCliWallet(acc.Address, "", ks)
}

func newCliWallet(owner common.Address, passphrase string, ks *keystore.KeyStore) (*CliWallet, error) {
	ehtClient, err := newEthClient()
	if err != nil {
		return nil, err
	}

	ownerAcc := accounts.Account{Address: owner}

	err = ks.Unlock(ownerAcc, passphrase)
	if err != nil {
		return nil, err
	}

	transactor := helpers.CreateNewKeystoreTransactor(ks, &ownerAcc)

	tokensContract, err := mysttoken.NewMystTokenTransactor(tokenAddress, ehtClient)

	paymentsContract, err := abigen.NewIdentityPromisesTransactor(paymentsAddress, ehtClient)
	if err != nil {
		return nil, err
	}

	return &CliWallet{
		txOpts:  transactor,
		Owner:   owner,
		backend: ehtClient,
		tokens: mysttoken.MystTokenTransactorSession{
			Contract:     tokensContract,
			TransactOpts: *transactor,
		},
		identityRegistry: abigen.IdentityPromisesTransactorSession{
			Contract:     paymentsContract,
			TransactOpts: *transactor,
		},
		ks: ks,
	}, nil
}

func initKeyStore(path string) *keystore.KeyStore {
	return keystore.NewKeyStore(path, keystore.StandardScryptN, keystore.StandardScryptP)
}

func registerIdentity(registrationData tequilapi_client.RegistrationDataDTO) error {
	defer os.RemoveAll("testdataoutput")

	//deployer account - owner of contracts, and can issue tokens
	masterAccWallet, err := NewDeployerWallet()
	if err != nil {
		return err
	}

	//random user
	userWallet, err := NewUserWallet("testdataoutput")
	if err != nil {
		return err
	}

	//user gets some ethers from master acc
	err = masterAccWallet.GiveEther(userWallet.Owner, 1, params.Ether)
	if err != nil {
		return err
	}

	//user buys some tokens in exchange
	err = masterAccWallet.GiveTokens(userWallet.Owner, 1000)
	if err != nil {
		return err
	}

	//user allows payments to take some tokens
	err = userWallet.ApproveForPayments(1000)
	if err != nil {
		return err
	}

	//user registers identity
	err = userWallet.RegisterIdentity(registrationData)
	return err
}

func topUpAccount(id string) error {
	//deployer account - owner of contracts, and can issue tokens
	masterAccWallet, err := NewDeployerWallet()
	if err != nil {
		return err
	}

	return masterAccWallet.GiveEther(common.HexToAddress(id), 1, params.Ether)
}
