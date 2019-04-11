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

	log "github.com/cihub/seelog"
	"github.com/pkg/errors"
)

func available(protocol string, port int) (bool, error) {
	addr := ":" + strconv.Itoa(port)

	switch protocol {
	case "udp", "udp4", "udp6":
		udpAddr, err := net.ResolveUDPAddr(protocol, addr)
		if err != nil {
			return false, errors.Wrap(err, "unable to resolve UDP address")
		}
		conn, err := net.ListenUDP(protocol, udpAddr)
		if err != nil {
			log.Infof("%s cannot listen on UDP port %v: %v", logPrefix, port, err)
			return false, nil
		}
		defer conn.Close()
	default:
		listener, err := net.Listen(protocol, addr)
		if err != nil {
			log.Infof("%s cannot listen on TCP port %v: %v", logPrefix, port, err)
			return false, nil
		}
		defer listener.Close()
	}

	return true, nil
}
