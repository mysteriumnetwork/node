/*
 * Copyright (C) 2023 The "MysteriumNetwork/node" Authors.
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

package sso

import (
	"encoding/json"
)

// MystnodesMessage expected by mystnodes.com
type MystnodesMessage struct {
	CodeChallenge string `json:"codeChallenge"`
	Identity      string `json:"identity"`
	RedirectURL   string `json:"redirectUrl"` // http://guillem.nodeUI
}

// JSON convenience receiver to convert MystnodesMessage struct to []byte
func (msg MystnodesMessage) JSON() ([]byte, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return []byte{}, err
	}
	return payload, nil
}
