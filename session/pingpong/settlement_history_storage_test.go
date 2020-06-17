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
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/stretchr/testify/assert"
)

func TestSettlementHistoryStorage(t *testing.T) {
	dir, err := ioutil.TempDir("", "providerInvoiceTest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	bolt, err := boltdb.NewStorage(dir)
	assert.NoError(t, err)
	defer bolt.Close()

	storage := NewSettlementHistoryStorage(bolt, 3)

	hermesID := common.HexToAddress("0x3313189b9b945DD38E7bfB6167F9909451582eE5")
	providerID := identity.FromAddress("0x79bb2a1c5E0075005F084a66A44D5e930A88eC86")
	entry := SettlementHistoryEntry{
		TxHash:       common.BigToHash(big.NewInt(123123123)),
		Promise:      crypto.Promise{},
		Beneficiary:  common.HexToAddress("0x4443189b9b945DD38E7bfB6167F9909451582eE5"),
		Amount:       big.NewInt(123),
		TotalSettled: big.NewInt(321),
	}

	t.Run("Returns not found if no results exist", func(t *testing.T) {
		entries, err := storage.Get(providerID, hermesID)
		assert.EqualError(t, err, errBoltNotFound)
		assert.Len(t, entries, 0)
	})

	t.Run("Inserts a history entry successfully", func(t *testing.T) {
		err := storage.Store(providerID, hermesID, entry)
		assert.NoError(t, err)
	})

	t.Run("Fetches the inserted entry", func(t *testing.T) {
		entries, err := storage.Get(providerID, hermesID)
		assert.NoError(t, err)
		assert.Len(t, entries, 1)

		assert.NotEqual(t, entry.Time, entries[0].Time)
		copy := entry
		copy.Time = entries[0].Time
		assert.EqualValues(t, copy, entries[0])
	})

	t.Run("Overrides old values if limit exceeded", func(t *testing.T) {
		err = storage.Store(providerID, hermesID, SettlementHistoryEntry{})
		assert.NoError(t, err)
		err = storage.Store(providerID, hermesID, SettlementHistoryEntry{})
		assert.NoError(t, err)
		err = storage.Store(providerID, hermesID, SettlementHistoryEntry{})
		assert.NoError(t, err)
		err = storage.Store(providerID, hermesID, SettlementHistoryEntry{})
		assert.NoError(t, err)

		entries, err := storage.Get(providerID, hermesID)
		assert.NoError(t, err)
		assert.Len(t, entries, 3)
	})

	t.Run("Returns sorted results", func(t *testing.T) {
		entries, err := storage.Get(providerID, hermesID)
		assert.NoError(t, err)
		assert.Len(t, entries, 3)

		assert.True(t, entries[0].Time.After(entries[1].Time))
		assert.True(t, entries[1].Time.After(entries[2].Time))
	})
}
