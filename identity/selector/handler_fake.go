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

package selector

import (
	"github.com/mysteriumnetwork/node/identity"
)

type handlerFake struct {
	id *identity.Identity
}

// NewFakeHandler fake mock handler
func NewFakeHandler() *handlerFake {
	return &handlerFake{}
}
func (hf *handlerFake) setIdentity(id *identity.Identity) {
	hf.id = id
}
func (hf *handlerFake) UseOrCreate(address, passphrase string) (identity.Identity, error) {
	if len(address) > 0 {
		return identity.Identity{Address: address}, nil
	}

	return identity.Identity{Address: "0x000000"}, nil
}
