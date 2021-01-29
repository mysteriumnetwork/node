/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package transport

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/rs/zerolog/log"
)

const sock = "/run/myst.sock"

// Start starts a listener on a unix domain socket.
// Conversation is handled by the handlerFunc.
func Start(handle handlerFunc, options Options) error {
	if err := os.RemoveAll(sock); err != nil {
		return fmt.Errorf("could not remove sock: %w", err)
	}
	l, err := net.Listen("unix", sock)
	if err != nil {
		return fmt.Errorf("error listening: %w", err)
	}
	numUid, err := strconv.Atoi(options.Uid)
	if err != nil {
		return fmt.Errorf("failed to parse uid %s: %w", options.Uid, err)
	}
	if err := os.Chown(sock, numUid, -1); err != nil {
		return fmt.Errorf("failed to chown supervisor socket to uid %s: %w", options.Uid, err)
	}
	if err := os.Chmod(sock, 0766); err != nil {
		return fmt.Errorf("failed to chmod supervisor socket: %w", err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			log.Err(err).Msg("Error closing listener")
		}
	}()
	log.Info().Msg("Waiting for connections...")
	for {
		conn, err := l.Accept()
		if err != nil {
			return fmt.Errorf("accept error: %w", err)
		}
		go func() {
			peer := conn.RemoteAddr().Network()
			log.Debug().Msgf("Client connected: %s", peer)
			handle(conn)
			if err := conn.Close(); err != nil {
				log.Err(err).Msgf("Error closing connection for: %v", peer)
			}
			log.Debug().Msgf("Client disconnected: %s", peer)
		}()
	}
}
