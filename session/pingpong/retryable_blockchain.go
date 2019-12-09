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
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/payments/bindings"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type blockchain interface {
	GetAccountantFee(accountantAddress common.Address) (uint16, error)
	IsRegisteredAsProvider(accountantAddress, registryAddress, addressToCheck common.Address) (bool, error)
	GetProviderChannel(accountantAddress common.Address, addressToCheck common.Address) (ProviderChannel, error)
	IsRegistered(registryAddress, addressToCheck common.Address) (bool, error)
	SubscribeToPromiseSettledEvent(providerID, accountantID common.Address) (sink chan *bindings.AccountantImplementationPromiseSettled, cancel func(), err error)
	GetConsumerBalance(channel, mystSCAddress common.Address) (*big.Int, error)
}

// BlockchainWithRetries takes in the plain blockchain implementation and exposes methods that will retry the underlying bc methods before giving up.
// This is required as the ethereum client will occasionally spit a TLS error if running for prolonged periods of time.
type BlockchainWithRetries struct {
	delay      time.Duration
	maxRetries int
	bc         blockchain
	stop       chan struct{}
	once       sync.Once
}

// ErrStopped represents an error when a call is interrupted
var ErrStopped = errors.New("call stopped")

// NewBlockchainWithRetries returns a new instance of blockchain with retries
func NewBlockchainWithRetries(bc blockchain, delay time.Duration, maxRetries int) *BlockchainWithRetries {
	return &BlockchainWithRetries{
		bc:         bc,
		delay:      delay,
		maxRetries: maxRetries,
	}
}

func (bwr *BlockchainWithRetries) callWithRetry(f func() error) error {
	for i := 0; i < bwr.maxRetries; i++ {
		err := f()
		if err == nil {
			return nil
		}
		if i+1 == bwr.maxRetries {
			return err
		}

		log.Warn().Err(err).Msgf("retry %v of %v", i+1, bwr.maxRetries)
		select {
		case <-bwr.stop:
			return ErrStopped
		case <-time.After(bwr.delay):
		}
	}
	return nil
}

// GetAccountantFee fetches the accountant fee from blockchain
func (bwr *BlockchainWithRetries) GetAccountantFee(accountantAddress common.Address) (uint16, error) {
	var res uint16
	err := bwr.callWithRetry(func() error {
		r, err := bwr.bc.GetAccountantFee(accountantAddress)
		if err != nil {
			return errors.Wrap(err, "could not get accountant fee")
		}
		res = r
		return nil
	})
	return res, err
}

// IsRegisteredAsProvider checks if the provider is registered with the accountant properly
func (bwr *BlockchainWithRetries) IsRegisteredAsProvider(accountantAddress, registryAddress, addressToCheck common.Address) (bool, error) {
	var res bool
	err := bwr.callWithRetry(func() error {
		r, err := bwr.bc.IsRegisteredAsProvider(accountantAddress, registryAddress, addressToCheck)
		if err != nil {
			return errors.Wrap(err, "could not check if registered as provider")
		}
		res = r
		return nil
	})
	return res, err
}

// GetProviderChannel returns the provider channel
func (bwr *BlockchainWithRetries) GetProviderChannel(accountantAddress, addressToCheck common.Address) (ProviderChannel, error) {
	var res ProviderChannel
	err := bwr.callWithRetry(func() error {
		r, err := bwr.bc.GetProviderChannel(accountantAddress, addressToCheck)
		if err != nil {
			return errors.Wrap(err, "could not get provider channel")
		}
		res = r
		return nil
	})

	return res, err
}

// SubscribeToPromiseSettledEvent subscribes to promise settled events
func (bwr *BlockchainWithRetries) SubscribeToPromiseSettledEvent(providerID, accountantID common.Address) (chan *bindings.AccountantImplementationPromiseSettled, func(), error) {
	var sink chan *bindings.AccountantImplementationPromiseSettled
	var cancel func()
	err := bwr.callWithRetry(func() error {
		s, c, err := bwr.bc.SubscribeToPromiseSettledEvent(providerID, accountantID)
		if err != nil {
			return errors.Wrap(err, "could not subscribe to settlement events")
		}
		sink = s
		cancel = c
		return nil
	})
	return sink, cancel, err
}

// IsRegistered checks wether the given identity is registered or not
func (bwr *BlockchainWithRetries) IsRegistered(registryAddress, addressToCheck common.Address) (bool, error) {
	var res bool
	err := bwr.callWithRetry(func() error {
		r, err := bwr.bc.IsRegistered(registryAddress, addressToCheck)
		if err != nil {
			return errors.Wrap(err, "check registration status")
		}
		res = r
		return nil
	})
	return res, err
}

// GetConsumerBalance returns the consumer balance in myst
func (bwr *BlockchainWithRetries) GetConsumerBalance(channel, mystSCAddress common.Address) (*big.Int, error) {
	var res *big.Int
	err := bwr.callWithRetry(func() error {
		result, bcErr := bwr.bc.GetConsumerBalance(channel, mystSCAddress)
		if bcErr != nil {
			return errors.Wrap(bcErr, "could not get consumer balance")
		}
		res = result
		return nil
	})
	return res, err
}

// Stop stops the blockhain with retries aborting any waits for retries
func (bwr *BlockchainWithRetries) Stop() {
	bwr.once.Do(func() {
		close(bwr.stop)
	})
}
