/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package tunnel

import "github.com/mysteriumnetwork/go-openvpn/openvpn/config"

// DefaultSetup represents a default tunnel setup - aka it sets the tun in configuration
type DefaultSetup struct {
}

// Setup implements the setup method for tunnel interface
func (ds *DefaultSetup) Setup(config *config.GenericConfig) error {
	config.SetDevice("tun")
	return nil
}

// Stop implements the stop method for tunnel interface
func (ds *DefaultSetup) Stop() {

}
