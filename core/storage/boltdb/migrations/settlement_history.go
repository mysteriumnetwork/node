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

package migrations

import (
	"math/big"
	"time"

	"github.com/asdine/storm/v3"
	"github.com/asdine/storm/v3/codec/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/payments/crypto"
	"go.etcd.io/bbolt"
)

type settlementEntryOld struct {
	Time         time.Time      `json:"time,omitempty"`
	TxHash       common.Hash    `json:"tx_hash,omitempty"`
	Promise      crypto.Promise `json:"promise,omitempty"`
	Beneficiary  common.Address `json:"beneficiary,omitempty"`
	Amount       *big.Int       `json:"amount,omitempty"`
	TotalSettled *big.Int       `json:"total_settled,omitempty"`
}

// Convert converts the old struct to a settlement struct
func (s settlementEntryOld) Convert() pingpong.SettlementHistoryEntry {
	return pingpong.SettlementHistoryEntry{
		TxHash:       s.TxHash,
		Time:         s.Time,
		Promise:      s.Promise,
		Beneficiary:  s.Beneficiary,
		Amount:       s.Amount,
		TotalSettled: s.TotalSettled,
	}
}

const settlementHistoryBucketOld = "settlement_history"
const settlementHistoryBucketNew = "settlement-history"

// SettlementValuesToRows converts settlement history from key-value struct to bucket with rows.
func SettlementValuesToRows(db *storm.DB) error {
	err := db.Bolt.View(func(tx *bbolt.Tx) error {
		oldBucket := tx.Bucket([]byte(settlementHistoryBucketOld))
		if oldBucket == nil {
			// Nothing to migrate
			return nil
		}

		newBucket := db.From(settlementHistoryBucketNew)

		return oldBucket.ForEach(func(k, v []byte) error {
			if string(k) == "__storm_metadata" {
				return nil
			}

			var oldEntries []settlementEntryOld
			if err := json.Codec.Unmarshal(v, &oldEntries); err != nil {
				return err
			}

			for _, oldEntry := range oldEntries {
				newEntry := oldEntry.Convert()
				if err := newBucket.Save(&newEntry); err != nil {
					return err
				}
			}

			return nil
		})
	})
	if err != nil {
		return err
	}

	return db.Bolt.Update(func(tx *bbolt.Tx) error {
		oldBucket := tx.Bucket([]byte(settlementHistoryBucketOld))
		if oldBucket == nil {
			// Nothing to migrate
			return nil
		}

		return tx.DeleteBucket([]byte(settlementHistoryBucketOld))
	})
}
