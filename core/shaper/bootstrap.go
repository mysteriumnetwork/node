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

package shaper

// Shaper shapes traffic on a network interface
type Shaper interface {
	// TargetInterface targets network interface for applying shaping configuration
	TargetInterface(interfaceName string)
	// Apply applies current shaping configuration to the target interface
	Apply() error
}

// Bootstrap creates a shaper: either a wondershaper (linux-only, if binary exists), or a noop
func Bootstrap() (shaper Shaper) {
	shaper, err := newWonderShaper()
	if err != nil {
		log.Error("Could not create wonder shaper, using noop: ", err)
		shaper = newNoopShaper()
	}
	return shaper
}
