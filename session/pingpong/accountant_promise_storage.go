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
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
)

const accountantPromiseBucketName = "accountant_promises"

// ErrAttemptToOverwrite occurs when a promise with lower value is attempted to be overwritten on top of an existing promise.
var ErrAttemptToOverwrite = errors.New("attempted to overwrite a promise with and equal or lower value")

// AccountantPromiseStorage allows for storing of accountant promises.
type AccountantPromiseStorage struct {
	lock sync.Mutex
	bolt persistentStorage
}

// NewAccountantPromiseStorage returns a new instance of the accountant promise storage.
func NewAccountantPromiseStorage(bolt persistentStorage) *AccountantPromiseStorage {
	return &AccountantPromiseStorage{
		bolt: bolt,
	}
}

// AccountantPromise represents a promise we store from the accountant
type AccountantPromise struct {
	Promise     crypto.Promise
	R           string
	Revealed    bool
	AgreementID uint64
}

// Store stores the given promise for the given accountant.
func (aps *AccountantPromiseStorage) Store(id identity.Identity, accountantID common.Address, promise AccountantPromise) error {
	aps.lock.Lock()
	defer aps.lock.Unlock()

	previousPromise, err := aps.get(id, accountantID)
	if err != nil && err != ErrNotFound {
		return err
	}

	if previousPromise.Promise.Amount >= promise.Promise.Amount {
		return ErrAttemptToOverwrite
	}

	channel, err := crypto.GenerateProviderChannelID(id.Address, accountantID.Hex())
	if err != nil {
		return errors.Wrap(err, "could not generate provider channel address")
	}

	return errors.Wrap(aps.bolt.SetValue(accountantPromiseBucketName, channel, promise), "could not store accountant promise")
}

func (aps *AccountantPromiseStorage) get(id identity.Identity, accountantID common.Address) (AccountantPromise, error) {
	channel, err := crypto.GenerateProviderChannelID(id.Address, accountantID.Hex())
	if err != nil {
		return AccountantPromise{}, errors.Wrap(err, "could not generate provider channel address")
	}

	result := &AccountantPromise{}
	err = aps.bolt.GetValue(accountantPromiseBucketName, channel, result)
	if err != nil {
		if err.Error() == errBoltNotFound {
			err = ErrNotFound
		} else {
			err = errors.Wrap(err, "could not get promise for accountant")
		}
	}
	return *result, err
}

// Get fetches the promise for the given accountant.
func (aps *AccountantPromiseStorage) Get(id identity.Identity, accountantID common.Address) (AccountantPromise, error) {
	aps.lock.Lock()
	defer aps.lock.Unlock()
	return aps.get(id, accountantID)
}
