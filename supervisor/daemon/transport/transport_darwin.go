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
	"log"
	"net"
	"os"
)

const sock = "/var/run/myst.sock"

// Start starts a listener on a unix domain socket.
// Conversation is handled by the handlerFunc.
func Start(handle handlerFunc) error {
	if err := os.RemoveAll(sock); err != nil {
		return fmt.Errorf("could not remove sock: %w", err)
	}
	l, err := net.Listen("unix", sock)
	if err != nil {
		return fmt.Errorf("error listening: %w", err)
	}
	// TODO: Change this permission and fix security. See https://github.com/mysteriumnetwork/node/issues/2204.
	if err := os.Chmod(sock, 0777); err != nil {
		return fmt.Errorf("failed to chmod the sock: %w", err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			log.Println("Error closing listener:", err)
		}
	}()
	for {
		log.Println("Waiting for connections...")
		conn, err := l.Accept()
		if err != nil {
			return fmt.Errorf("accept error: %w", err)
		}
		go func() {
			peer := conn.RemoteAddr().Network()
			log.Println("Client connected:", peer)
			handle(conn)
			if err := conn.Close(); err != nil {
				log.Printf("Error closing connection for: %v error: %v", peer, err)
			}
			log.Println("Client disconnected:", peer)
		}()
	}
}
