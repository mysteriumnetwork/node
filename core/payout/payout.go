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

package payout

import (
	"time"

	"github.com/asdine/storm/v3"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/mmn"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	bucket = "payout-address-bucket"
)

// ErrInvalidAddress represents invalid address error
var (
	ErrInvalidAddress = errors.New("invalid address")
	ErrNotFound       = errors.New("beneficiary not found")
)

type storage interface {
	Store(bucket string, data interface{}) error
	GetOneByField(bucket string, fieldName string, key interface{}, to interface{}) error
}

type mmnClient interface {
	UpdateBeneficiary(data *mmn.UpdateBeneficiaryRequest) error
	GetBeneficiary(identityStr string) (string, error)
}

// AddressStorage handles storing of payout address
type AddressStorage struct {
	storage storage
	mmnc    mmnClient
}

// NewAddressStorage constructor
func NewAddressStorage(storage storage, mmnc mmnClient) *AddressStorage {
	return &AddressStorage{
		storage: storage,
		mmnc:    mmnc,
	}
}

// Save save payout address for identity
func (as *AddressStorage) Save(identity, address string) error {
	if !common.IsHexAddress(address) {
		return ErrInvalidAddress
	}
	err := as.mmnc.UpdateBeneficiary(&mmn.UpdateBeneficiaryRequest{
		Beneficiary: address,
		Identity:    identity,
	})
	if err != nil {
		return err
	}

	store := &storedBeneficiary{
		ID:          identity,
		Beneficiary: address,
		LastUpdated: time.Now().UTC(),
	}
	return as.storage.Store(bucket, store)
}

// Address retrieve payout address for identity
func (as *AddressStorage) Address(identity string) (string, error) {
	result := &storedBeneficiary{}
	err := as.storage.GetOneByField(bucket, "ID", identity, result)
	if err != nil && err != storm.ErrNotFound {
		return "", err
	}

	if err == storm.ErrNotFound || result.isExpired() {
		addr, err := as.mmnc.GetBeneficiary(identity)
		if err != nil {
			return "", err
		}
		if addr == "" {
			return "", ErrNotFound
		}
		if err := as.Save(identity, addr); err != nil {
			log.Warn().Err(err).Msg("could not save benefiicary after not finding it in local cache, will try next time")
		}

		return addr, nil
	}
	return result.Beneficiary, nil
}

type storedBeneficiary struct {
	ID          string `storm:"id"`
	Beneficiary string
	LastUpdated time.Time
}

func (s *storedBeneficiary) isExpired() bool {
	if s.LastUpdated.IsZero() {
		return true
	}
	now := time.Now().UTC().AddDate(0, 0, 1)
	return s.LastUpdated.After(now)
}
