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

package port

import (
	"net"
	"strconv"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Tests port by opening a UDP listener on given port number
func available(port int) (bool, error) {
	addr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(port))
	if err != nil {
		return false, errors.Wrap(err, "unable to resolve UDP address")
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Info().Err(err).Msgf("Cannot listen on UDP port %d", port)
		return false, nil
	}
	defer conn.Close()

	return true, nil
}
