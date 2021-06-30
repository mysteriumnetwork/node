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
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

const (
	bucket = "payout-address-bucket"
)

// ErrInvalidAddress represents invalid address error
var ErrInvalidAddress = errors.New("Invalid address")

type storage interface {
	GetValue(bucket string, key interface{}, to interface{}) error
	SetValue(bucket string, key interface{}, to interface{}) error
}

// AddressStorage handles storing of payout address
type AddressStorage struct {
	storage storage
}

// NewAddressStorage constructor
func NewAddressStorage(storage storage) *AddressStorage {
	return &AddressStorage{storage: storage}
}

// Save save payout address for identity
func (as *AddressStorage) Save(identity, address string) error {
	if !common.IsHexAddress(address) {
		return ErrInvalidAddress
	}
	return as.storage.SetValue(bucket, identity, address)
}

// Address retrieve payout address for identity
func (as *AddressStorage) Address(identity string) (string, error) {
	var addr string
	return addr, as.storage.GetValue(bucket, identity, &addr)
}
