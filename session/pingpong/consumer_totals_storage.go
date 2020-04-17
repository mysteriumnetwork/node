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
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/pkg/errors"
)

const consumerTotalStorageBucketName = "consumer_promised_totals"

// ConsumerTotalsStorage allows to store total promised amounts for each channel.
type ConsumerTotalsStorage struct {
	bolt persistentStorage
	lock sync.Mutex
	bus  eventbus.Publisher
}

// NewConsumerTotalsStorage creates a new instance of consumer totals storage.
func NewConsumerTotalsStorage(bolt persistentStorage, bus eventbus.Publisher) *ConsumerTotalsStorage {
	return &ConsumerTotalsStorage{
		bolt: bolt,
		bus:  bus,
	}
}

// Store stores the given amount as promised for the given channel.
func (cts *ConsumerTotalsStorage) Store(id identity.Identity, accountantID common.Address, amount uint64) error {
	cts.lock.Lock()
	defer cts.lock.Unlock()

	err := cts.bolt.SetValue(consumerTotalStorageBucketName, id.Address+accountantID.Hex(), amount)
	if err != nil {
		return err
	}

	go cts.bus.Publish(event.AppTopicGrandTotalChanged, event.AppEventGrandTotalChanged{
		Current:      amount,
		AccountantID: accountantID,
		ConsumerID:   id,
	})

	return nil
}

// Get fetches the amount as promised for the given channel.
func (cts *ConsumerTotalsStorage) Get(id identity.Identity, accountantID common.Address) (uint64, error) {
	cts.lock.Lock()
	defer cts.lock.Unlock()
	var res uint64
	err := cts.bolt.GetValue(consumerTotalStorageBucketName, id.Address+accountantID.Hex(), &res)
	if err != nil {
		// wrap the error to an error we can check for
		if err.Error() == errBoltNotFound {
			err = ErrNotFound
		} else {
			err = errors.Wrap(err, "could not get total promised")
		}
	}
	return res, err
}
