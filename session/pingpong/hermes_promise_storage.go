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

	"github.com/asdine/storm/v3/codec/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
	"go.etcd.io/bbolt"
)

const hermesPromiseBucketName = "hermes_promises"

// ErrAttemptToOverwrite occurs when a promise with lower value is attempted to be overwritten on top of an existing promise.
var ErrAttemptToOverwrite = errors.New("attempted to overwrite a promise with and equal or lower value")

// HermesPromiseStorage allows for storing of hermes promises.
type HermesPromiseStorage struct {
	lock sync.Mutex
	bolt *boltdb.Bolt
}

// NewHermesPromiseStorage returns a new instance of the hermes promise storage.
func NewHermesPromiseStorage(bolt *boltdb.Bolt) *HermesPromiseStorage {
	return &HermesPromiseStorage{
		bolt: bolt,
	}
}

// HermesPromise represents a promise we store from the hermes
type HermesPromise struct {
	ChannelID   string
	Identity    identity.Identity
	HermesID    common.Address
	Promise     crypto.Promise
	R           string
	Revealed    bool
	AgreementID *big.Int
}

// Store stores the given promise.
func (aps *HermesPromiseStorage) Store(promise HermesPromise) error {
	aps.lock.Lock()
	defer aps.lock.Unlock()

	previousPromise, err := aps.get(promise.ChannelID)
	if err != nil && err != ErrNotFound {
		return err
	}

	if promise.Promise.Amount == nil {
		promise.Promise.Amount = big.NewInt(0)
	}

	if previousPromise.Promise.Amount != nil && previousPromise.Promise.Amount.Cmp(promise.Promise.Amount) >= 0 {
		return ErrAttemptToOverwrite
	}

	return errors.Wrap(aps.bolt.SetValue(hermesPromiseBucketName, promise.ChannelID, promise), "could not store hermes promise")
}

func (aps *HermesPromiseStorage) get(channelID string) (HermesPromise, error) {
	result := &HermesPromise{}
	err := aps.bolt.GetValue(hermesPromiseBucketName, channelID, result)
	if err != nil {
		if err.Error() == errBoltNotFound {
			err = ErrNotFound
		} else {
			err = errors.Wrap(err, "could not get promise for hermes")
		}
	}
	return *result, err
}

// Get fetches the promise by channel ID identifier.
func (aps *HermesPromiseStorage) Get(channelID string) (HermesPromise, error) {
	aps.lock.Lock()
	defer aps.lock.Unlock()
	return aps.get(channelID)
}

// HermesPromiseFilter defines all flags for filtering in promises in storage.
type HermesPromiseFilter struct {
	Identity *identity.Identity
	HermesID *common.Address
}

// List fetches the promise for the given hermes.
func (aps *HermesPromiseStorage) List(filter HermesPromiseFilter) ([]HermesPromise, error) {
	aps.lock.Lock()
	defer aps.lock.Unlock()

	result := make([]HermesPromise, 0)
	err := aps.bolt.DB().Bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(hermesPromiseBucketName))
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			if string(k) == "__storm_metadata" {
				return nil
			}

			var entry HermesPromise
			if err := json.Codec.Unmarshal(v, &entry); err != nil {
				return err
			}

			if filter.Identity != nil {
				if *filter.Identity != entry.Identity {
					return nil
				}
			}
			if filter.HermesID != nil {
				if *filter.HermesID != entry.HermesID {
					return nil
				}
			}

			result = append(result, entry)
			return nil
		})
	})
	return result, err
}
