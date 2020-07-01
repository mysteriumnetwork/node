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

	"github.com/rs/zerolog/log"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows/svc"
)

const sock = `\\.\pipe\mystpipe`

// Start starts a listener on a unix domain socket.
// Conversation is handled by the handlerFunc.
func Start(handle handlerFunc, options Options) error {
	if options.WinService {
		return svc.Run("MysteriumVPNSupervisor", &managerService{handle: handle})
	} else {
		return listenPipe(handle)
	}
}

type managerService struct {
	handle handlerFunc
}

// Execute is an entrypoint for a windows service.
func (m *managerService) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue

	s <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	go func() {
		if err := listenPipe(m.handle); err != nil {
			log.Err(err).Msgf("Could not listen pipe on %s", sock)
		}
	}()

	for c := range r {
		switch c.Cmd {
		case svc.Interrogate:
			s <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			return
		case svc.Pause:
			s <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
		case svc.Continue:
			s <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
		default:
			log.Error().Msgf("Unexpected control request #%d", c)
		}
	}
	return
}

func listenPipe(handle handlerFunc) error {
	// https://support.microsoft.com/en-us/help/243330/well-known-security-identifiers-in-windows-operating-systems
	// Looking up by name appears to be unreliable: the name could not be found in localized installations.
	// Using a well-known security identifier here instead.
	usersSID := "S-1-5-32-545"
	sddl := "D:P(A;;GA;;;BA)(A;;GA;;;SY)"
	sddl += fmt.Sprintf("(A;;GRGW;;;%s)", usersSID)
	c := winio.PipeConfig{
		SecurityDescriptor: sddl,
		MessageMode:        true,
		InputBufferSize:    65536,
		OutputBufferSize:   65536,
	}

	l, err := winio.ListenPipe(sock, &c)
	if err != nil {
		return fmt.Errorf("error listening: %w", err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			log.Err(err).Msg("Error closing listener")
		}
	}()
	for {
		log.Debug().Msg("Waiting for connections...")
		conn, err := l.Accept()
		if err != nil {
			return fmt.Errorf("accept error: %w", err)
		}
		go func() {
			peer := conn.RemoteAddr().Network()
			log.Debug().Msgf("Client connected: %s", peer)
			handle(conn)
			if err := conn.Close(); err != nil {
				log.Err(err).Msgf("Error closing connection for: %s", peer)
			}
			log.Debug().Msgf("Client disconnected: %s", peer)
		}()
	}
}
