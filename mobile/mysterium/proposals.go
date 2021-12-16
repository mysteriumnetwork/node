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

import "encoding/json"

// GetCountries returns service proposals number per country from API.
// go mobile does not support complex slices.
func (mb *MobileNode) GetCountries(req *GetProposalsRequest) ([]byte, error) {
	countries, err := mb.proposalsManager.getCountries(req)
	if err != nil {
		return nil, err
	}
	return json.Marshal(countries)
}

// GetProposals returns service proposals from API or cache. Proposals returned as JSON byte array since
// go mobile does not support complex slices.
func (mb *MobileNode) GetProposals(req *GetProposalsRequest) ([]byte, error) {
	proposals, err := mb.proposalsManager.getProposals(req)
	if err != nil {
		return nil, err
	}
	return json.Marshal(proposals)
}

// GetProposalsByPreset returns service proposals by presetID.
func (mb *MobileNode) GetProposalsByPreset(presetID int) ([]byte, error) {
	proposals, err := mb.proposalsManager.getProposals(&GetProposalsRequest{PresetID: presetID})
	if err != nil {
		return nil, err
	}
	return json.Marshal(proposals)
}
