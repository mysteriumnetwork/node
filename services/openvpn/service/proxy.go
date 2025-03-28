/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package service

import (
	"fmt"
	"io"
	"net"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/p2p"
)

func proxyOpenVPN(conn p2p.ServiceConn, serverPort int) error {
	localPort, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("127.0.0.1:%d", serverPort))
	if err != nil {
		return err
	}

	openVPNProxy, err := net.DialUDP("udp", nil, localPort)
	if err != nil {
		return err
	}

	go copyStreams(openVPNProxy, conn)
	go copyStreams(conn, openVPNProxy)

	return nil
}

func copyStreams(dstConn p2p.ServiceConn, srcConn p2p.ServiceConn) {
	const bufferLen = 2048 * 1024
	buf := make([]byte, bufferLen)

	defer dstConn.Close()
	defer srcConn.Close()

	dc, ok := dstConn.(*net.UDPConn)
	if !ok {
		log.Error().Msg("Failed to cast dstConn to *net.UDPConn")
		return
	}

	sc, ok := srcConn.(*net.UDPConn)
	if !ok {
		log.Error().Msg("Failed to cast srcConn to *net.UDPConn")
		return
	}

	_, err := io.CopyBuffer(dc, sc, buf)
	if err != nil {
		log.Error().Msg("Failed to write/read a stream to/from service natProxy")
	}
}
