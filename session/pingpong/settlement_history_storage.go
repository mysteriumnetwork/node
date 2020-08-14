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
	"time"

	"github.com/asdine/storm/v3/q"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
)

// SettlementHistoryStorage stores the settlement events for historical purposes.
type SettlementHistoryStorage struct {
	bolt *boltdb.Bolt
}

// NewSettlementHistoryStorage returns a new instance of the SettlementHistoryStorage.
func NewSettlementHistoryStorage(bolt *boltdb.Bolt) *SettlementHistoryStorage {
	return &SettlementHistoryStorage{
		bolt: bolt,
	}
}

// SettlementHistoryEntry represents a settlement history entry
type SettlementHistoryEntry struct {
	TxHash         common.Hash       `json:"tx_hash,omitempty" storm:"id"`
	ProviderID     identity.Identity `json:"provider_id,omitempty"`
	AccountantID   common.Address    `json:"accountant_id,omitempty"`
	ChannelAddress common.Address    `json:"channel_address,omitempty"`
	Time           time.Time         `json:"time,omitempty"`
	Promise        crypto.Promise    `json:"promise,omitempty"`
	Beneficiary    common.Address    `json:"beneficiary,omitempty"`
	Amount         *big.Int          `json:"amount,omitempty"`
	TotalSettled   *big.Int          `json:"total_settled,omitempty"`
}

const settlementHistoryBucket = "settlement-history"

// Store stores a given settlement history entry.
func (shs *SettlementHistoryStorage) Store(she SettlementHistoryEntry) error {
	return shs.bolt.DB().From(settlementHistoryBucket).Save(&she)
}

// Get returns the settlement history for given provider accountant combination.
func (shs *SettlementHistoryStorage) Get(provider identity.Identity, accountant common.Address) ([]SettlementHistoryEntry, error) {
	query := shs.bolt.DB().
		From(settlementHistoryBucket).
		Select(
			q.Eq("ProviderID", provider),
			q.Eq("AccountantID", accountant),
		).
		OrderBy("Time").
		Reverse()

	var res []SettlementHistoryEntry
	return res, query.Find(&res)
}
