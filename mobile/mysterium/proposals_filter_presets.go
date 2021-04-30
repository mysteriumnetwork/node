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

import (
	"encoding/json"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
)

// ProposalFilterPreset represents proposal filter preset
type ProposalFilterPreset struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ListProposalFilterPresets lists system and user created filter presets for proposals
// see ProposalFilterPreset for contract
func (mb *MobileNode) ListProposalFilterPresets() ([]byte, error) {
	list, err := mb.filterPresetStorage.List()
	if err != nil {
		return nil, err
	}

	entries := list.Entries
	result := make([]ProposalFilterPreset, len(entries))
	for i, entry := range list.Entries {
		result[i] = preset(entry)
	}
	return json.Marshal(&result)
}

func preset(entry proposal.FilterPreset) ProposalFilterPreset {
	return ProposalFilterPreset{
		ID:   entry.ID,
		Name: entry.Name,
	}
}
