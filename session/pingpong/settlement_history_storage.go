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
	"errors"
	"math/big"
	"time"

	"github.com/asdine/storm/v3"
	"github.com/asdine/storm/v3/q"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
)

// HistoryType settlement history type
// swagger:model HistoryType
type HistoryType string

const (
	// SettlementType settlement type
	SettlementType HistoryType = "settlement"
	// WithdrawalType withdrawal type
	WithdrawalType HistoryType = "withdrawal"
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
	TxHash           common.Hash `storm:"id"`
	BlockExplorerURL string
	ProviderID       identity.Identity
	HermesID         common.Address
	ChannelAddress   common.Address
	Time             time.Time
	Promise          crypto.Promise
	Beneficiary      common.Address
	Amount           *big.Int
	TotalSettled     *big.Int
	Fees             *big.Int
	IsWithdrawal     bool
	Error            string
}

const settlementHistoryBucket = "settlement-history"

// Store stores a given settlement history entry.
func (shs *SettlementHistoryStorage) Store(she SettlementHistoryEntry) error {
	shs.bolt.Lock()
	defer shs.bolt.Unlock()
	return shs.bolt.DB().From(settlementHistoryBucket).Save(&she)
}

// SettlementHistoryFilter defines all flags for filtering in settlement history storage.
type SettlementHistoryFilter struct {
	TimeFrom   *time.Time
	TimeTo     *time.Time
	ProviderID *identity.Identity
	HermesID   *common.Address
	Types      []HistoryType
}

// List retrieves stored entries.
func (shs *SettlementHistoryStorage) List(filter SettlementHistoryFilter) (result []SettlementHistoryEntry, err error) {
	where := make([]q.Matcher, 0)
	if filter.TimeFrom != nil {
		where = append(where, q.Gte("Time", filter.TimeFrom.UTC()))
	}
	if filter.TimeTo != nil {
		where = append(where, q.Lte("Time", filter.TimeTo.UTC()))
	}
	if filter.ProviderID != nil {
		id := filter.ProviderID
		where = append(where, q.Eq("ProviderID", *id))
	}
	if filter.HermesID != nil {
		where = append(where, q.Eq("HermesID", filter.HermesID))
	}
	if len(filter.Types) > 0 {
		if contains(filter.Types, WithdrawalType) {
			where = append(where, q.Eq("IsWithdrawal", true))
		}
		if contains(filter.Types, SettlementType) {
			where = append(where, q.Eq("IsWithdrawal", false))
		}
	}

	shs.bolt.RLock()
	defer shs.bolt.RUnlock()
	sq := shs.bolt.DB().
		From(settlementHistoryBucket).
		Select(q.And(where...)).
		OrderBy("Time").
		Reverse()

	err = sq.Find(&result)
	if errors.Is(err, storm.ErrNotFound) {
		return []SettlementHistoryEntry{}, nil
	}

	return result, err
}

func contains(sources []HistoryType, target HistoryType) bool {
	for _, source := range sources {
		if source == target {
			return true
		}
	}

	return false
}
