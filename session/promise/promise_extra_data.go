/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package promise

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mysteriumnetwork/node/session/promise/model"
)

// ExtraData represents the extra data in the promise
type ExtraData struct {
	ConsumerAddress common.Address
}

// Hash returns the hash of the extra data
func (extra ExtraData) Hash() []byte {
	return crypto.Keccak256(extra.ConsumerAddress.Bytes())
}

var _ model.ExtraData = ExtraData{}
