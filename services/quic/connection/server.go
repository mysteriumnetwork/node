/*
 * Copyright (C) 2025 The "MysteriumNetwork/node" Authors.
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

package connection

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/services/quic/streams"
)

// Server represents QUIC server.
type Server struct {
	transportConn quic.Connection
	addrServe     string
	basicUser     string
	basicPassword string

	trafficIn  uint64
	trafficOut uint64

	l net.Listener
}

// NewServer creates new QUIC server.
func NewServer(transportConn quic.Connection, addrServe, basicUser, basicPassword string) *Server {
	return &Server{
		addrServe:     addrServe,
		transportConn: transportConn,
		basicUser:     basicUser,
		basicPassword: basicPassword,
	}
}

func (s *Server) listenAndServeRequests(ctx context.Context) (err error) {
	s.l, err = net.Listen("tcp4", s.addrServe)
	if err != nil {
		return fmt.Errorf("failed to listen on TCP address: %w", err)
	}

	go func() {
		if err := http.Serve(s.l, s); err != nil {
			log.Error().Err(err).Msg("failed to serve HTTP")
		}
	}()

	return nil
}

// ServeHTTP serves HTTP requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.validateCredentials(r) {
		http.Error(w, "Invalid credentials", http.StatusProxyAuthRequired)
		return
	}

	if _, err := w.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
		return
	}

	src, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		log.Error().Err(err).Msg("failed to hijack connection")
		return
	}

	stream, err := s.transportConn.OpenStream()
	if err != nil {
		return
	}

	defer stream.Close()

	req := &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Host: r.RequestURI},
		Host:   r.RequestURI,
		Header: r.Header,
	}

	err = req.Write(stream)
	if err != nil {
		log.Error().Err(err).Msg("failed to write request")
	}

	resp, err := http.ReadResponse(bufio.NewReader(stream), req)
	if err != nil {
		log.Error().Err(err).Msg("failed to read response")
		return
	}
	defer resp.Body.Close()

	if err := stream.SetDeadline(time.Time{}); err != nil {
		log.Error().Err(err).Msg("failed to reset deadline to 0")
		log.Error().Msgf("failed to reset deadline to 0: %s", err)
		return
	}

	if resp.StatusCode != 200 {
		log.Error().Msgf("failed to do connect handshake, status code: %s, endpoint: %s", resp.Status, r.RequestURI)
		return
	}

	streams.ConnectStreams(r.Context(), src, stream, s.updateStats)
}

func (s *Server) updateStats(direction string, bytes uint64) {
	switch direction {
	case "Upload":
		atomic.AddUint64(&s.trafficIn, uint64(bytes))
	case "Download":
		atomic.AddUint64(&s.trafficOut, uint64(bytes))
	}
}

// Stop stops the server.
func (s *Server) Stop() {
	if s.l != nil {
		s.l.Close()
	}
}

// Stats returns server statistics.
func (s *Server) Stats() (uint64, uint64) {
	return atomic.LoadUint64(&s.trafficIn), atomic.LoadUint64(&s.trafficOut)
}
