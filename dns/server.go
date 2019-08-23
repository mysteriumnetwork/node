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

package dns

import (
	"github.com/miekg/dns"
)

// NewServer returns new instance of API server
func NewServer(addr string, handler dns.Handler) *Server {
	server := new(Server)
	server.Addr = addr
	server.dnsServer = &dns.Server{
		Addr:    addr,
		Net:     "udp",
		Handler: handler,
	}
	return server
}

// Server defines DNS server with all handler attached to it
type Server struct {
	Addr      string
	dnsServer *dns.Server
}

// Run starts DNS server
func (server *Server) Run() error {
	return server.dnsServer.ListenAndServe()
}

// Stop shutdowns Proxy server
func (server *Server) Stop() error {
	return server.dnsServer.Shutdown()
}
