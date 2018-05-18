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

type identityCacheFake struct {
	identity Identity
}

// NewIdentityCacheFake creates and returns fake identity cache
func NewIdentityCacheFake() IdentityCacheInterface {
	return &identityCacheFake{}
}

// GetIdentity returns mocked identity
func (icf *identityCacheFake) GetIdentity() (identity Identity, err error) {
	return icf.identity, nil
}

// StoreIdentity saves identity to be retrieved later
func (icf *identityCacheFake) StoreIdentity(identity Identity) error {
	icf.identity = identity
	return nil
}
