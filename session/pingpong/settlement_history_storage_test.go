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
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/stretchr/testify/assert"
)

func TestSettlementHistoryStorage(t *testing.T) {
	dir, err := os.MkdirTemp("", "providerInvoiceTest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	bolt, err := boltdb.NewStorage(dir)
	assert.NoError(t, err)
	defer bolt.Close()

	storage := NewSettlementHistoryStorage(bolt)

	hermesAddress := common.HexToAddress("0x3313189b9b945DD38E7bfB6167F9909451582eE5")
	providerID := identity.FromAddress("0x79bb2a1c5E0075005F084a66A44D5e930A88eC86")
	entry1 := SettlementHistoryEntry{
		TxHash:       common.BigToHash(big.NewInt(1)),
		ProviderID:   providerID,
		HermesID:     hermesAddress,
		Time:         time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC),
		Promise:      crypto.Promise{},
		Beneficiary:  common.HexToAddress("0x4443189b9b945DD38E7bfB6167F9909451582eE5"),
		Amount:       big.NewInt(123),
		TotalSettled: big.NewInt(321),
	}
	entry2 := SettlementHistoryEntry{
		TxHash:       common.BigToHash(big.NewInt(2)),
		ProviderID:   providerID,
		HermesID:     hermesAddress,
		Time:         time.Date(2020, 1, 1, 2, 0, 0, 0, time.UTC),
		Promise:      crypto.Promise{},
		Beneficiary:  common.HexToAddress("0x4443189b9b945DD38E7bfB6167F9909451582eE5"),
		Amount:       big.NewInt(456),
		TotalSettled: big.NewInt(654),
	}

	t.Run("Returns empty list if no results exist", func(t *testing.T) {
		entries, err := storage.List(SettlementHistoryFilter{})
		assert.NoError(t, err)
		assert.Len(t, entries, 0)
		assert.EqualValues(t, []SettlementHistoryEntry{}, entries)
	})

	t.Run("Inserts a history entry successfully", func(t *testing.T) {
		err := storage.Store(entry1)
		assert.NoError(t, err)
	})

	t.Run("Fetches the inserted entry", func(t *testing.T) {
		entries, err := storage.List(SettlementHistoryFilter{})
		assert.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.EqualValues(t, []SettlementHistoryEntry{entry1}, entries)
	})

	t.Run("Returns sorted results", func(t *testing.T) {
		err := storage.Store(entry2)
		assert.NoError(t, err)

		entries, err := storage.List(SettlementHistoryFilter{})
		assert.NoError(t, err)
		assert.Len(t, entries, 2)
		assert.EqualValues(t, []SettlementHistoryEntry{entry2, entry1}, entries)
	})
}
