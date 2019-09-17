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

	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
)

const consumerTotalStorageBucketName = "consumer_promised_totals"

// ConsumerTotalsStorage allows to store total promised amounts for each channel
type ConsumerTotalsStorage struct {
	bolt persistentStorage
	lock sync.Mutex
}

// NewConsumerTotalsStorage creates a new instance of consumer totals storage
func NewConsumerTotalsStorage(bolt persistentStorage) *ConsumerTotalsStorage {
	return &ConsumerTotalsStorage{
		bolt: bolt,
	}
}

// Store stores the given amount as promised for the given channel
func (cts *ConsumerTotalsStorage) Store(channelAddress string, amount uint64) error {
	cts.lock.Lock()
	defer cts.lock.Unlock()
	return cts.bolt.SetValue(consumerTotalStorageBucketName, channelAddress, amount)
}

// ChannelAddressParams contains all the params required for channel address generation
type ChannelAddressParams struct {
	Identity, Registry, ChannelImplementation string
}

// ToChannelAddress converts the channel address params to a channel address
func (cap ChannelAddressParams) ToChannelAddress() (string, error) {
	addr, err := crypto.GenerateChannelAddress(cap.Identity, cap.Registry, cap.ChannelImplementation)
	return addr, errors.Wrap(err, "could not generate channel address")
}

// GenerateAndStore generates the channel address and stores the given amount as promised
func (cts *ConsumerTotalsStorage) GenerateAndStore(cap ChannelAddressParams, amount uint64) error {
	addr, err := cap.ToChannelAddress()
	if err != nil {
		return err
	}
	return cts.Store(addr, amount)
}

// Get fetches the amount as promised for the given channel
func (cts *ConsumerTotalsStorage) Get(channelAddress string) (uint64, error) {
	cts.lock.Lock()
	defer cts.lock.Unlock()
	var res uint64
	err := cts.bolt.GetValue(consumerTotalStorageBucketName, channelAddress, &res)
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
