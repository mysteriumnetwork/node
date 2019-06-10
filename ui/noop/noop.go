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

package noop

import "sync"

// Server doesn't do much really
type Server struct {
	iSimulateWork chan error
	once          sync.Once
}

// NewServer returns a new noop server
func NewServer() *Server {
	return &Server{
		iSimulateWork: make(chan error),
	}
}

// Serve blocks
func (s *Server) Serve() error {
	return <-s.iSimulateWork
}

// Stop stops the blocking of serve
func (s *Server) Stop() {
	s.once.Do(func() {
		close(s.iSimulateWork)
	})
}
