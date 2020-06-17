/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
)

// SettlementHistoryStorage stores the settlement events for historical purposes.
type SettlementHistoryStorage struct {
	bolt                 persistentStorage
	maxEntriesPerChannel int
	lock                 sync.Mutex
}

// DefaultMaxEntriesPerChannel represents the default settlement history limit.
const DefaultMaxEntriesPerChannel = 100

// NewSettlementHistoryStorage returns a new instance of the SettlementHistoryStorage.
func NewSettlementHistoryStorage(bolt persistentStorage, maxEntries int) *SettlementHistoryStorage {
	return &SettlementHistoryStorage{
		bolt:                 bolt,
		maxEntriesPerChannel: maxEntries,
	}
}

// SettlementHistoryEntry represents a settlement history entry
type SettlementHistoryEntry struct {
	Time         time.Time      `json:"time,omitempty"`
	TxHash       common.Hash    `json:"tx_hash,omitempty"`
	Promise      crypto.Promise `json:"promise,omitempty"`
	Beneficiary  common.Address `json:"beneficiary,omitempty"`
	Amount       *big.Int       `json:"amount,omitempty"`
	TotalSettled *big.Int       `json:"total_settled,omitempty"`
}

const settlementHistoryBucket = "settlement_history"

// Store sotres a given settlement history entry.
func (shs *SettlementHistoryStorage) Store(provider identity.Identity, hermes common.Address, she SettlementHistoryEntry) error {
	shs.lock.Lock()
	defer shs.lock.Unlock()
	addr, err := crypto.GenerateProviderChannelID(provider.Address, hermes.Hex())
	if err != nil {
		return fmt.Errorf("could not generate provider channel address: %w", err)
	}

	she.Time = time.Now().UTC()
	toStore := []SettlementHistoryEntry{she}
	entries, err := shs.get(addr)
	if err != nil {
		if err.Error() == errBoltNotFound {
			// if not found, just store and return
			return shs.store(addr, toStore)
		}
		return fmt.Errorf("could not get previous settlement history: %w", err)
	}

	// sort by date
	sort.Slice(entries, func(i, j int) bool { return entries[i].Time.After(entries[j].Time) })

	//  remove old ones if needed to prevent excessive history storage
	if len(entries) < shs.maxEntriesPerChannel {
		toStore = append(toStore, entries...)
	} else if len(entries) == shs.maxEntriesPerChannel {
		toStore = append(toStore, entries[:len(entries)-1]...)
	} else {
		diff := len(entries) - shs.maxEntriesPerChannel + 1
		toStore = append(toStore, entries[:len(entries)-diff]...)
	}

	return shs.store(addr, toStore)
}

func (shs *SettlementHistoryStorage) store(channel string, she []SettlementHistoryEntry) error {
	err := shs.bolt.SetValue(settlementHistoryBucket, channel, she)
	if err != nil {
		return fmt.Errorf("could not store settlement history: %w", err)
	}
	return nil
}

// Get returns the settlement history for given provider hermes combination.
func (shs *SettlementHistoryStorage) Get(provider identity.Identity, hermes common.Address) ([]SettlementHistoryEntry, error) {
	shs.lock.Lock()
	defer shs.lock.Unlock()
	addr, err := crypto.GenerateProviderChannelID(provider.Address, hermes.Hex())
	if err != nil {
		return []SettlementHistoryEntry{}, fmt.Errorf("could not generate provider channel address: %w", err)
	}
	return shs.get(addr)
}

func (shs *SettlementHistoryStorage) get(channel string) ([]SettlementHistoryEntry, error) {
	var res []SettlementHistoryEntry
	return res, shs.bolt.GetValue(settlementHistoryBucket, channel, &res)
}
