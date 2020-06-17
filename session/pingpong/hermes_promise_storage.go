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

const hermesPromiseBucketName = "hermes_promises"

// ErrAttemptToOverwrite occurs when a promise with lower value is attempted to be overwritten on top of an existing promise.
var ErrAttemptToOverwrite = errors.New("attempted to overwrite a promise with and equal or lower value")

// HermesPromiseStorage allows for storing of hermes promises.
type HermesPromiseStorage struct {
	lock sync.Mutex
	bolt persistentStorage
}

// NewHermesPromiseStorage returns a new instance of the hermes promise storage.
func NewHermesPromiseStorage(bolt persistentStorage) *HermesPromiseStorage {
	return &HermesPromiseStorage{
		bolt: bolt,
	}
}

// HermesPromise represents a promise we store from the hermes
type HermesPromise struct {
	Promise     crypto.Promise
	R           string
	Revealed    bool
	AgreementID uint64
}

// Store stores the given promise for the given hermes.
func (aps *HermesPromiseStorage) Store(id identity.Identity, hermesID common.Address, promise HermesPromise) error {
	aps.lock.Lock()
	defer aps.lock.Unlock()

	previousPromise, err := aps.get(id, hermesID)
	if err != nil && err != ErrNotFound {
		return err
	}

	if previousPromise.Promise.Amount >= promise.Promise.Amount {
		return ErrAttemptToOverwrite
	}

	channel, err := crypto.GenerateProviderChannelID(id.Address, hermesID.Hex())
	if err != nil {
		return errors.Wrap(err, "could not generate provider channel address")
	}

	return errors.Wrap(aps.bolt.SetValue(hermesPromiseBucketName, channel, promise), "could not store hermes promise")
}

func (aps *HermesPromiseStorage) get(id identity.Identity, hermesID common.Address) (HermesPromise, error) {
	channel, err := crypto.GenerateProviderChannelID(id.Address, hermesID.Hex())
	if err != nil {
		return HermesPromise{}, errors.Wrap(err, "could not generate provider channel address")
	}

	result := &HermesPromise{}
	err = aps.bolt.GetValue(hermesPromiseBucketName, channel, result)
	if err != nil {
		if err.Error() == errBoltNotFound {
			err = ErrNotFound
		} else {
			err = errors.Wrap(err, "could not get promise for hermes")
		}
	}
	return *result, err
}

// Get fetches the promise for the given hermes.
func (aps *HermesPromiseStorage) Get(id identity.Identity, hermesID common.Address) (HermesPromise, error) {
	aps.lock.Lock()
	defer aps.lock.Unlock()
	return aps.get(id, hermesID)
}
