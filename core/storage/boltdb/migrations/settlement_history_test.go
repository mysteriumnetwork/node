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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/storage/boltdb/boltdbtest"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/payments/crypto"

	"github.com/stretchr/testify/assert"
)

var (
	oldSettlementMock = settlementEntryOld{
		TxHash:       common.BigToHash(big.NewInt(123123123)),
		Promise:      crypto.Promise{},
		Beneficiary:  common.HexToAddress("0x4443189b9b945DD38E7bfB6167F9909451582eE5"),
		Amount:       big.NewInt(123),
		TotalSettled: big.NewInt(321),
	}
)

func Test_ToSettlementRows_WithNoData(t *testing.T) {
	// given
	file, db := boltdbtest.CreateDB(t)
	defer boltdbtest.CleanupDB(t, file, db)

	// when
	err := SettlementValuesToRows(db)
	assert.NoError(t, err)
}

func Test_ToSettlementRows_WithData(t *testing.T) {
	// given
	file, db := boltdbtest.CreateDB(t)
	defer boltdbtest.CleanupDB(t, file, db)

	err := db.Set(settlementHistoryBucketOld, "1", []settlementEntryOld{oldSettlementMock})
	assert.NoError(t, err)

	// when
	err = SettlementValuesToRows(db)
	assert.NoError(t, err)

	// then
	var entries []pingpong.SettlementHistoryEntry
	err = db.From(settlementHistoryBucketNew).All(&entries)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, []pingpong.SettlementHistoryEntry{oldSettlementMock.Convert()}, entries)
}
