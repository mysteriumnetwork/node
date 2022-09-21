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
	"fmt"
	"math/big"
	"sync"

	"github.com/asdine/storm/v3"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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
func (cts *ConsumerTotalsStorage) Store(chainID int64, id identity.Identity, hermesID common.Address, amount *big.Int) error {
	cts.lock.Lock()
	defer cts.lock.Unlock()

	key := cts.makeKey(chainID, id, hermesID)
	err := cts.bolt.SetValue(consumerTotalStorageBucketName, key, amount)
	if err != nil {
		return err
	}

	go cts.bus.Publish(event.AppTopicGrandTotalChanged, event.AppEventGrandTotalChanged{
		ChainID:    chainID,
		Current:    amount,
		HermesID:   hermesID,
		ConsumerID: id,
	})

	return nil
}

// Get fetches the amount as promised for the given channel.
func (cts *ConsumerTotalsStorage) Get(chainID int64, id identity.Identity, hermesID common.Address) (*big.Int, error) {
	cts.lock.Lock()
	defer cts.lock.Unlock()
	var res = new(big.Int)
	key := cts.makeKey(chainID, id, hermesID)
	err := cts.bolt.GetValue(consumerTotalStorageBucketName, key, &res)
	if err != nil {
		// wrap the error to an error we can check for
		if errors.Is(storm.ErrNotFound, err) {
			err = ErrNotFound
		} else {
			err = errors.Wrap(err, "could not get total promised")
		}
	}
	return res, err
}

// Add adds the given amount as promised for the given channel, if none exists it adds from 0.
func (cts *ConsumerTotalsStorage) Add(chainID int64, id identity.Identity, hermesID common.Address, amount *big.Int) error {
	cts.lock.Lock()
	defer cts.lock.Unlock()

	key := cts.makeKey(chainID, id, hermesID)
	value := new(big.Int)
	err := cts.bolt.GetValue(consumerTotalStorageBucketName, key, &value)
	if err != nil {
		// wrap the error to an error we can check for
		if errors.Is(storm.ErrNotFound, err) {
			log.Debug().Msg("No previous invoice grand total, assuming zero")
			value = big.NewInt(0)
		} else {
			return fmt.Errorf("could not get total promised to add : %w", err)
		}
	}
	if value == nil || value.Cmp(big.NewInt(0)) == -1 {
		reason := "nil"
		if value != nil {
			reason = "negative"
		}
		log.Warn().Msgf("got %s consumer total to add from db, assuming 0", reason)
		value = big.NewInt(0)
	}
	totalAmount := new(big.Int).Add(value, amount)
	err = cts.bolt.SetValue(consumerTotalStorageBucketName, key, totalAmount)
	if err != nil {
		return err
	}

	go cts.bus.Publish(event.AppTopicGrandTotalChanged, event.AppEventGrandTotalChanged{
		ChainID:    chainID,
		Current:    totalAmount,
		HermesID:   hermesID,
		ConsumerID: id,
	})

	return nil
}

func (cts *ConsumerTotalsStorage) makeKey(chainID int64, id identity.Identity, hermesID common.Address) string {
	return fmt.Sprintf("%d%s%s", chainID, id.Address, hermesID.Hex())
}
