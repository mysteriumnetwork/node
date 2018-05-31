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

import "github.com/mysterium/node/identity"

// Selector selects the identity
type Selector func() (identity.Identity, error)

// LoadIdentity chooses which identity to use and invokes it using identityHandler
func LoadIdentity(identityHandler HandlerInterface, identityOption, passphrase string) (identity.Identity, error) {
	if len(identityOption) > 0 {
		return identityHandler.UseExisting(identityOption, passphrase)
	}

	if id, err := identityHandler.UseLast(passphrase); err == nil {
		return id, err
	}

	return identityHandler.UseNew(passphrase)
}
