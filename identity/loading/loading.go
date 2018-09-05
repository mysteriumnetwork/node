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

import "github.com/mysteriumnetwork/node/identity"

// Loader selects the identity
type Loader func() (identity.Identity, error)

// NewLoader chooses which identity to use and invokes it using identityHandler
func NewLoader(identityHandler Handler, identityOption, passphrase string) Loader {
	return func() (identity.Identity, error) {
		if len(identityOption) > 0 {
			return identityHandler.UseExisting(identityOption, passphrase)
		}

		if id, err := identityHandler.UseLast(passphrase); err == nil {
			return id, err
		}

		return identityHandler.UseNew(passphrase)
	}
}
