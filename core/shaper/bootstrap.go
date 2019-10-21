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

import (
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/rs/zerolog/log"
)

// Shaper shapes traffic on a network interface
type Shaper interface {
	// Start applies current shaping configuration for the specified interface
	// and then continuously ensures it
	Start(interfaceName string) error
}

// Bootstrap creates a shaper: either a wondershaper (linux-only, if binary exists), or a noop
func Bootstrap(eventBus eventbus.EventBus) (shaper Shaper) {
	shaper, err := newWonderShaper(eventBus)
	if err != nil {
		log.Warn().Err(err).Msg("Could not create wonder shaper, using noop")
		shaper = newNoopShaper()
	}
	return shaper
}
