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

	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
)

// ConsumerTotalsStorage allows to store total promised amounts for each channel.
type ConsumerTotalsStorage struct {
	createLock sync.RWMutex
	bus        eventbus.Publisher
	data       map[string]*ConsumerTotalElement
}

// ConsumerTotalElement stores a grand total promised amount for a single identity, hermes and chain id
type ConsumerTotalElement struct {
	lock   sync.RWMutex
	amount *big.Int
}

// NewConsumerTotalsStorage creates a new instance of consumer totals storage.
func NewConsumerTotalsStorage(bus eventbus.Publisher) *ConsumerTotalsStorage {
	return &ConsumerTotalsStorage{
		bus:  bus,
		data: make(map[string]*ConsumerTotalElement),
	}
}

// Store stores the given amount as promised for the given channel.
func (cts *ConsumerTotalsStorage) Store(chainID int64, id identity.Identity, hermesID common.Address, amount *big.Int) error {
	cts.createLock.Lock()
	defer cts.createLock.Unlock()

	key := cts.makeKey(chainID, id, hermesID)
	_, ok := cts.data[key]
	if !ok {
		_, ok := cts.data[key]
		if !ok {
			cts.data[key] = &ConsumerTotalElement{
				amount: nil,
			}
		}
	}
	element, ok := cts.data[key]
	if !ok {
		return fmt.Errorf("key was not created properly")
	}
	element.lock.Lock()
	defer element.lock.Unlock()
	if element.amount != nil && element.amount.Cmp(amount) == 1 {
		log.Warn().Fields(map[string]interface{}{
			"old_value": element.amount.String(),
			"new_value": amount.String(),
			"identity":  id.Address,
			"chain_id":  chainID,
			"hermes_id": hermesID.Hex(),
		}).Msg("tried to save a lower grand total amount")
		return nil
	}
	element.amount = amount

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
	key := cts.makeKey(chainID, id, hermesID)
	cts.createLock.RLock()
	defer cts.createLock.RUnlock()

	element, ok := cts.data[key]
	if !ok {
		return nil, ErrNotFound
	}
	element.lock.RLock()
	defer element.lock.RUnlock()
	res := element.amount
	if res == nil {
		return nil, ErrNotFound
	}
	return res, nil
}

// Add adds the given amount as promised for the given channel.
func (cts *ConsumerTotalsStorage) Add(chainID int64, id identity.Identity, hermesID common.Address, amount *big.Int) error {
	cts.createLock.Lock()
	defer cts.createLock.Unlock()

	key := cts.makeKey(chainID, id, hermesID)
	_, ok := cts.data[key]
	if !ok {
		_, ok := cts.data[key]
		if !ok {
			cts.data[key] = &ConsumerTotalElement{
				amount: nil,
			}
		}
	}
	element, ok := cts.data[key]
	if !ok {
		return fmt.Errorf("key was not created properly")
	}
	element.lock.Lock()
	defer element.lock.Unlock()
	oldAmount := element.amount
	if oldAmount == nil {
		oldAmount = big.NewInt(0)
	}
	newAmount := new(big.Int).Add(oldAmount, amount)
	element.amount = newAmount

	go cts.bus.Publish(event.AppTopicGrandTotalChanged, event.AppEventGrandTotalChanged{
		ChainID:    chainID,
		Current:    newAmount,
		HermesID:   hermesID,
		ConsumerID: id,
	})
	return nil
}

func (cts *ConsumerTotalsStorage) makeKey(chainID int64, id identity.Identity, hermesID common.Address) string {
	return fmt.Sprintf("%d%s%s", chainID, id.Address, hermesID.Hex())
}
