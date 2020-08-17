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

	"github.com/asdine/storm/v3"
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
	TxHash         common.Hash `storm:"id"`
	ProviderID     identity.Identity
	AccountantID   common.Address
	ChannelAddress common.Address
	Time           time.Time
	Promise        crypto.Promise
	Beneficiary    common.Address
	Amount         *big.Int
	TotalSettled   *big.Int
}

const settlementHistoryBucket = "settlement-history"

// Store stores a given settlement history entry.
func (shs *SettlementHistoryStorage) Store(she SettlementHistoryEntry) error {
	return shs.bolt.DB().From(settlementHistoryBucket).Save(&she)
}

// Query executes given query.
func (shs *SettlementHistoryStorage) Query(query *SettlementHistoryQuery) (err error) {
	return query.run(shs.bolt.DB().From(settlementHistoryBucket))
}

// NewSettlementHistoryQuery creates instance of query.
func NewSettlementHistoryQuery() *SettlementHistoryQuery {
	return &SettlementHistoryQuery{}
}

// SettlementHistoryQuery defines all flags for filtering in settlement history storage.
type SettlementHistoryQuery struct {
	Entries []SettlementHistoryEntry

	filterFrom       *time.Time
	filterTo         *time.Time
	filterProvider   *identity.Identity
	filterAccountant *common.Address
}

// FilterFrom filters fetched sessions from given time.
func (qr *SettlementHistoryQuery) FilterFrom(from time.Time) *SettlementHistoryQuery {
	from = from.UTC()
	qr.filterFrom = &from
	return qr
}

// FilterTo filters fetched sessions to given time.
func (qr *SettlementHistoryQuery) FilterTo(to time.Time) *SettlementHistoryQuery {
	to = to.UTC()
	qr.filterTo = &to
	return qr
}

// FilterProviderID filters fetched entries by provider ID.
func (qr *SettlementHistoryQuery) FilterProviderID(providerID identity.Identity) *SettlementHistoryQuery {
	qr.filterProvider = &providerID
	return qr
}

// FilterAccountantID filters fetched entries by accountant ID.
func (qr *SettlementHistoryQuery) FilterAccountantID(accountantID common.Address) *SettlementHistoryQuery {
	qr.filterAccountant = &accountantID
	return qr
}

// FetchEntries fetches list of sessions to Query.Entries.
func (qr *SettlementHistoryQuery) FetchEntries() *SettlementHistoryQuery {
	return qr
}

func (qr *SettlementHistoryQuery) run(node storm.Node) error {
	where := make([]q.Matcher, 0)
	if qr.filterFrom != nil {
		where = append(where, q.Gte("Time", qr.filterFrom))
	}
	if qr.filterTo != nil {
		where = append(where, q.Lte("Time", qr.filterTo))
	}
	if qr.filterProvider != nil {
		where = append(where, q.Eq("ProviderID", qr.filterProvider))
	}
	if qr.filterAccountant != nil {
		where = append(where, q.Eq("AccountantID", qr.filterAccountant))
	}

	sq := node.
		Select(q.And(where...)).
		OrderBy("Time").
		Reverse()

	err := sq.Find(&qr.Entries)
	if err == storm.ErrNotFound {
		qr.Entries = []SettlementHistoryEntry{}
		return nil
	}
	return err
}
