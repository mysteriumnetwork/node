/*
 * Copyright (C) 2023 The "MysteriumNetwork/node" Authors.
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
	"strings"
	"time"

	"github.com/asdine/storm/v3"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

const (
	bucket = "beneficiary-address"
)

// ErrInvalidAddress represents invalid address error
var (
	ErrInvalidAddress = errors.New("invalid address")
	ErrNotFound       = errors.New("beneficiary not found")
)

type localStorage interface {
	Store(bucket string, data interface{}) error
	GetOneByField(bucket string, fieldName string, key interface{}, to interface{}) error
}

// BeneficiaryStorage handles storing of beneficiary address
type BeneficiaryStorage interface {
	Address(identity string) (string, error)
	Save(identity, address string) error
}

// AddressStorage handles storing of beneficiary address
type AddressStorage struct {
	storage localStorage
}

// NewAddressStorage constructor
func NewAddressStorage(storage localStorage) *AddressStorage {
	return &AddressStorage{
		storage: storage,
	}
}

// Save beneficiary address for identity
func (as *AddressStorage) Save(identity, address string) error {
	if !common.IsHexAddress(address) {
		return ErrInvalidAddress
	}

	store := &storedBeneficiary{
		ID:          strings.ToLower(identity),
		Beneficiary: address,
		LastUpdated: time.Now().UTC(),
	}
	return as.storage.Store(bucket, store)
}

// Address retrieve beneficiary address for identity
func (as *AddressStorage) Address(identity string) (string, error) {
	result := &storedBeneficiary{}
	err := as.storage.GetOneByField(bucket, "ID", strings.ToLower(identity), result)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return "", ErrNotFound
		}
		return "", err
	}

	return result.Beneficiary, nil
}

type storedBeneficiary struct {
	ID          string `storm:"id"`
	Beneficiary string
	LastUpdated time.Time
}
