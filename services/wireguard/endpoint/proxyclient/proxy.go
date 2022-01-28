/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package proxyclient

import (
	"io"
	"net"
	"net/http"
	"net/http/httputil"

	"github.com/rs/zerolog/log"
	netproxy "golang.org/x/net/proxy"
)

type proxyServer struct {
	dialer netproxy.Dialer
	ln     net.Listener
}

func newProxy(upstreamDialer netproxy.Dialer, ln net.Listener) *proxyServer {
	return &proxyServer{
		dialer: upstreamDialer,
		ln:     ln,
	}
}

func (s *proxyServer) Serve() error {
	for {
		c, err := s.ln.Accept()
		if err != nil {
			return err
		}
		go func() {
			s.handle(c)
			c.Close()
		}()
	}
}

func (s *proxyServer) handle(c net.Conn) {
	sc := httputil.NewServerConn(c, nil)
	req, err := sc.Read()
	if err != nil {
		log.Error().Err(err).Msg("Failed to read HTTP request")
		return
	}

	conn, err := s.dialer.Dial("tcp", req.Host)
	if err != nil {
		log.Error().Err(err).Msgf("Error establishing connection to %s", req.Host)
		return
	}
	defer conn.Close()

	if req.Method == http.MethodConnect {
		c.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))
	} else if err := req.Write(conn); err != nil {
		log.Error().Err(err).Msgf("Failed to forward HTTP request to %s", req.Host)
		return
	}

	go io.Copy(conn, c)
	io.Copy(c, conn)
}
