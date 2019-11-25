/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package pingpong

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mysteriumnetwork/payments/bindings"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ProviderChannel represents the provider channel
type ProviderChannel struct {
	Beneficiary   common.Address
	Balance       *big.Int
	Settled       *big.Int
	Loan          *big.Int
	LastUsedNonce *big.Int
	Timelock      *big.Int
}

// Blockchain contains all the useful blockchain utilities for the payment off chain messaging
type Blockchain struct {
	client    *ethclient.Client
	bcTimeout time.Duration
}

// NewBlockchain returns a new instance of blockchain
func NewBlockchain(c *ethclient.Client, timeout time.Duration) *Blockchain {
	return &Blockchain{
		client:    c,
		bcTimeout: timeout,
	}
}

// GetAccountantFee fetches the accountant fee from blockchain
func (bc *Blockchain) GetAccountantFee(accountantAddress common.Address) (uint16, error) {
	caller, err := bindings.NewAccountantImplementationCaller(accountantAddress, bc.client)
	if err != nil {
		return 0, errors.Wrap(err, "could not create accountant implementation caller")
	}

	ctx, cancel := context.WithTimeout(context.Background(), bc.bcTimeout)
	defer cancel()

	res, err := caller.LastFee(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return 0, errors.Wrap(err, "could not get accountant fee")
	}

	return res.Value, err
}

// IsRegisteredAsProvider checks if the provider is registered with the accountant properly
func (bc *Blockchain) IsRegisteredAsProvider(accountantAddress, registryAddress, addressToCheck common.Address) (bool, error) {
	registered, err := bc.IsRegistered(registryAddress, addressToCheck)
	if err != nil {
		return false, errors.Wrap(err, "could not check registration status")
	}

	if !registered {
		return false, nil
	}

	res, err := bc.getProviderChannelLoan(accountantAddress, addressToCheck)
	if err != nil {
		return false, errors.Wrap(err, "could not get provider channel loan amount")
	}

	return res.Cmp(big.NewInt(0)) == 1, nil
}

// GetProviderChannel returns the provider channel
func (bc *Blockchain) GetProviderChannel(accountantAddress common.Address, addressToCheck common.Address) (ProviderChannel, error) {
	addressBytes, err := bc.getProviderChannelAddressBytes(accountantAddress, addressToCheck)
	if err != nil {
		return ProviderChannel{}, errors.Wrap(err, "could not calculate provider channel address")
	}
	log.Info().Msgf("checkign address %v", common.Bytes2Hex(addressBytes[:]))
	caller, err := bindings.NewAccountantImplementationCaller(accountantAddress, bc.client)
	if err != nil {
		return ProviderChannel{}, errors.Wrap(err, "could not create accountant caller")
	}

	ctx, cancel := context.WithTimeout(context.Background(), bc.bcTimeout)
	defer cancel()

	ch, err := caller.Channels(&bind.CallOpts{
		Context: ctx,
	}, addressBytes)
	return ch, errors.Wrap(err, "could not get provider channel from bc")
}

func (bc *Blockchain) getProviderChannelLoan(accountantAddress common.Address, addressToCheck common.Address) (*big.Int, error) {
	ch, err := bc.GetProviderChannel(accountantAddress, addressToCheck)
	return ch.Loan, errors.Wrap(err, "could not get provider channel from bc")
}

func (bc *Blockchain) getProviderChannelAddressBytes(accountantAddress, addressToCheck common.Address) ([32]byte, error) {
	addressBytes := [32]byte{}

	addr, err := crypto.GenerateProviderChannelID(addressToCheck.Hex(), accountantAddress.Hex())
	if err != nil {
		return addressBytes, errors.Wrap(err, "could not generate channel address")
	}

	copy(addressBytes[:], crypto.Pad(common.Hex2Bytes(strings.TrimPrefix(addr, "0x")), 32))

	return addressBytes, nil
}

// SubscribeToPromiseSettledEvent subscribes to promise settled events
func (bc *Blockchain) SubscribeToPromiseSettledEvent(providerID, accountantID common.Address) (sink chan *bindings.AccountantImplementationPromiseSettled, cancel func(), err error) {
	caller, err := bindings.NewAccountantImplementationFilterer(accountantID, bc.client)
	if err != nil {
		return sink, cancel, errors.Wrap(err, "could not create accountant caller")
	}
	sink = make(chan *bindings.AccountantImplementationPromiseSettled)
	addr, err := bc.getProviderChannelAddressBytes(accountantID, providerID)
	if err != nil {
		return sink, cancel, errors.Wrap(err, "could not get provider channel address")
	}
	sub, err := caller.WatchPromiseSettled(&bind.WatchOpts{}, sink, [][32]byte{addr})
	if err != nil {
		return sink, cancel, errors.Wrap(err, "could not subscribe to promise settlement events")
	}

	go func() {
		subErr := <-sub.Err()
		if subErr != nil {
			log.Error().Err(err).Msg("subscription error")
		}
		close(sink)
	}()

	return sink, sub.Unsubscribe, nil
}

// IsRegistered checks wether the given identity is registered or not
func (bc *Blockchain) IsRegistered(registryAddress, addressToCheck common.Address) (bool, error) {
	caller, err := bindings.NewRegistryCaller(registryAddress, bc.client)
	if err != nil {
		return false, errors.Wrap(err, "could not create registry caller")
	}

	ctx, cancel := context.WithTimeout(context.Background(), bc.bcTimeout)
	defer cancel()

	res, err := caller.IsRegistered(&bind.CallOpts{
		Context: ctx,
	}, addressToCheck)
	return res, errors.Wrap(err, "could not check registration status")
}
