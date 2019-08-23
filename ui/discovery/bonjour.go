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

package discovery

import (
	"github.com/oleksandr/bonjour"
)

type bonjourServer struct {
	port   int
	server *bonjour.Server
}

func newBonjourServer(uiPort int) *bonjourServer {
	return &bonjourServer{
		port: uiPort,
	}
}

func (bs *bonjourServer) Start() (err error) {
	bs.server, err = bonjour.Register("Mysterium Node", "_mysterium-node._tcp", "", bs.port, nil, nil)
	return err
}

func (bs *bonjourServer) Stop() (err error) {
	if bs.server != nil {
		bs.server.Shutdown()
	}
	return nil
}
