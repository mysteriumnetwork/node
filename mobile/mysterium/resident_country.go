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

package mysterium

// ResidentCountryUpdateRequest represents a partial typed config (as appose to map[string]interface blob)
type ResidentCountryUpdateRequest struct {
	IdentityAddress string
	Country         string
}

// ResidentCountry resident country
func (mb *MobileNode) ResidentCountry() string {
	return mb.residentCountry.Get()
}

// UpdateResidentCountry updates resident country
func (mb *MobileNode) UpdateResidentCountry(r *ResidentCountryUpdateRequest) error {
	return mb.residentCountry.Save(r.IdentityAddress, r.Country)
}
