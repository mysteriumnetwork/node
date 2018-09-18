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

import (
	"errors"

	"github.com/mysteriumnetwork/go-openvpn/openvpn/config"
)

const tunLogPrefix = "[linux tun service] "

// ErrNoFreeTunDevice is thrown when no free tun device is available on system
var ErrNoFreeTunDevice = errors.New("no free tun device found")

// Setup represents the operations required for a tunnel setup
type Setup interface {
	Setup(config *config.GenericConfig) error
	Stop()
}
