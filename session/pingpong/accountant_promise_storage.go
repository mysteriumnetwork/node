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

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
)

const accountantPromiseBucketName = "accountant_promises"

// AccountantPromiseStorage allows for storing of accountant promises
type AccountantPromiseStorage struct {
	lock sync.Mutex
	bolt persistentStorage
}

// NewAccountantPromiseStorage returns a new instance of the accountant promise storage
func NewAccountantPromiseStorage(bolt persistentStorage) *AccountantPromiseStorage {
	return &AccountantPromiseStorage{
		bolt: bolt,
	}
}

// Store stores the given promise for the given accountant
func (aps *AccountantPromiseStorage) Store(accountantID identity.Identity, promise crypto.Promise) error {
	aps.lock.Lock()
	defer aps.lock.Unlock()
	return errors.Wrap(aps.bolt.SetValue(accountantPromiseBucketName, accountantID.Address, promise), "could not store accountant promise")
}

// Get fetches the promise for the given accountant
func (aps *AccountantPromiseStorage) Get(accountantID identity.Identity) (crypto.Promise, error) {
	aps.lock.Lock()
	defer aps.lock.Unlock()
	result := &crypto.Promise{}
	err := aps.bolt.GetValue(accountantPromiseBucketName, accountantID.Address, result)
	if err != nil {
		if err.Error() == errBoltNotFound {
			err = ErrNotFound
		} else {
			err = errors.Wrap(err, "could not get promise for accountant")
		}
	}
	return *result, err
}
