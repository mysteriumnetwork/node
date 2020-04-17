/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package identity

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// Identity represents unique user network identity
type Identity struct {
	// TODO Encoding should be in transport layer
	Address string `json:"address"`
}

// ToCommonAddress returns the common address representation for identity
func (i Identity) ToCommonAddress() common.Address {
	return common.HexToAddress(i.Address)
}

// FromAddress converts address to identity
func FromAddress(address string) Identity {
	return Identity{
		Address: strings.ToLower(address),
	}
}
