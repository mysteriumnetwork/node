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

package loading

import (
	"errors"

	"github.com/mysteriumnetwork/node/identity"
)

type handlerFake struct {
	LastAddress string
}

func (hf *handlerFake) UseExisting(address, passphrase string) (identity.Identity, error) {
	return identity.Identity{Address: address}, nil
}

func (hf *handlerFake) UseLast(passphrase string) (id identity.Identity, err error) {
	if hf.LastAddress != "" {
		id = identity.Identity{Address: hf.LastAddress}
	} else {
		err = errors.New("no last identity")
	}
	return
}

func (hf *handlerFake) UseNew(passphrase string) (identity.Identity, error) {
	return identity.Identity{Address: "new"}, nil
}
