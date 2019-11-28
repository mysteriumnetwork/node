/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

import "github.com/mysteriumnetwork/node/core/node"

type embeddedLibCheck struct {
}

// Check always returns nil as embedded lib does not have any external failing deps
func (embeddedLibCheck) Check() error {
	return nil
}

// BinaryPath returns noop binary path
func (embeddedLibCheck) BinaryPath() string {
	return "mobile uses embedded openvpn lib"
}

// check if our struct satisfies Openvpn interface expected by node options
var _ node.Openvpn = embeddedLibCheck{}
