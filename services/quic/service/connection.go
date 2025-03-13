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

package service

import (
	"context"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/services/quic/streams"
)

var (
	_ net.Listener = &listener{}
	_ net.Conn     = &conn{}
)

type listener struct {
	ctx context.Context
	c   quic.Connection
}

func (l *listener) Accept() (net.Conn, error) {
	s, err := l.c.AcceptStream(l.ctx)
	if err != nil {
		return nil, err
	}

	return &conn{
		Stream: s,
		local:  l.c.LocalAddr(),
		remote: l.c.RemoteAddr(),
	}, nil
}

func (l *listener) Close() error {
	return l.c.CloseWithError(100, "closed")
}

func (l *listener) Addr() net.Addr {
	return nil
}

type conn struct {
	quic.Stream
	local, remote net.Addr
}

func (l *conn) LocalAddr() net.Addr {
	return l.local
}

func (l *conn) RemoteAddr() net.Addr {
	return l.remote
}

type connectServer struct {
	connectResponse []byte

	trafficIn  uint64
	trafficOut uint64
}

func (c *connectServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodConnect {
		http.Error(w, "only CONNECT requests allowed", http.StatusMethodNotAllowed)
		log.Error().Msg("Only CONNECT requests allowed")
		return
	}

	dst, err := net.Dial("tcp", r.RequestURI)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if _, err := w.Write(c.connectResponse); err != nil {
		return
	}

	src, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	streams.ConnectStreams(r.Context(), src, dst, c.updateStats)
}

func (c *connectServer) updateStats(direction string, bytes uint64) {
	switch direction {
	case "Upload":
		atomic.AddUint64(&c.trafficIn, uint64(bytes))
	case "Download":
		atomic.AddUint64(&c.trafficOut, uint64(bytes))
	}
}

func (c *connectServer) Stats() (uint64, uint64) {
	return atomic.LoadUint64(&c.trafficIn), atomic.LoadUint64(&c.trafficOut)
}
