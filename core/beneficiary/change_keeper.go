/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package beneficiary

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/rs/zerolog/log"
)

// SettleState represents the state of settle with beneficiary transaction
type SettleState string

const (
	// Pending transaction is pending
	Pending SettleState = "pending"
	// Completed transaction is completed
	Completed SettleState = "completed"
	// NotFound transaction state
	NotFound SettleState = "not_found"

	bucketChangeStatus = "beneficiary-change-status"
)

type beneficiaryChangeKeeper struct {
	st      storage
	chainID int64
}

func newBeneficiaryChangeKeeper(chainID int64, st storage) *beneficiaryChangeKeeper {
	return &beneficiaryChangeKeeper{
		chainID: chainID,
		st:      st,
	}
}

// ChangeStatus holds Beneficiary settlement transaction information
type ChangeStatus struct {
	ChangeTo string
	State    SettleState
	Error    string
}

func newChangeStatus(changeTo string, state SettleState, err error) *ChangeStatus {
	n := &ChangeStatus{
		ChangeTo: changeTo,
		State:    state,
		Error:    "",
	}

	if err != nil {
		n.Error = err.Error()
	}

	return n
}

func (bcs *beneficiaryChangeKeeper) updateChangeStatus(id identity.Identity, sbs SettleState, beneficiary string, settleError error) (*ChangeStatus, error) {
	entry := newChangeStatus(beneficiary, sbs, settleError)
	err := bcs.st.SetValue(bucketChangeStatus, storageKey(bcs.chainID, id.Address), entry)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (bcs *beneficiaryChangeKeeper) executeWithStatusTracking(id identity.Identity, bf common.Address, fn func() error) error {
	if _, err := bcs.updateChangeStatus(id, Pending, bf.Hex(), nil); err != nil {
		return fmt.Errorf("failed to start tracking status change: %w", err)
	}
	callbackErr := fn()

	if _, err := bcs.updateChangeStatus(id, Completed, bf.Hex(), callbackErr); err != nil {
		log.Err(err).Msg("saving beneficiary change status")
	}

	return callbackErr
}

// GetChangeStatus returns current change transaction status.
func (bcs *beneficiaryChangeKeeper) GetChangeStatus(id identity.Identity) (*ChangeStatus, error) {
	var b ChangeStatus
	if err := bcs.st.GetValue(bucketChangeStatus, storageKey(bcs.chainID, id.Address), &b); err != nil {
		return nil, err
	}
	return &b, nil
}

// CleanupAndGetChangeStatus cleans up current change status and returns it.
//
// Cleanup is done using the provided currentBeneficiary, if the beneficiaries match
// and transaction is still in state "Pending" it will be bumped to completed, else
// nothing is done.
func (bcs *beneficiaryChangeKeeper) CleanupAndGetChangeStatus(id identity.Identity, currentBeneficiary string) (*ChangeStatus, error) {
	b, err := bcs.GetChangeStatus(id)
	if err != nil {
		return nil, err
	}

	// Not pending, no need to cleanup.
	if b.State != Pending {
		return b, nil
	}

	if b.ChangeTo != currentBeneficiary {
		return b, nil
	}

	return bcs.updateChangeStatus(id, Completed, currentBeneficiary, nil)
}
