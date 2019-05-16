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

package web

import "sync"

// NoopServer doesn't do much really
type NoopServer struct {
	iSimulateWork chan error
	once          sync.Once
}

// NewNoopServer returns a new noop server
func NewNoopServer() *NoopServer {
	return &NoopServer{
		iSimulateWork: make(chan error),
	}
}

// Serve blocks
func (s *NoopServer) Serve() error {
	return <-s.iSimulateWork
}

// Stop stops the blocking of serve
func (s *NoopServer) Stop() {
	s.once.Do(func() {
		close(s.iSimulateWork)
	})
}
