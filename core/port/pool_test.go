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

package port

import (
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAcquiredPortsAreUsable(t *testing.T) {
	// given
	pool := NewPool()

	// when
	tcpPort, _ := pool.Acquire("tcp")
	// then
	err := listenTcp(tcpPort.Num())
	assert.NoError(t, err)

	// when
	udpPort, _ := pool.Acquire("udp")
	// then
	err = listenUdp(udpPort.Num())
	assert.NoError(t, err)
}

func listenTcp(port int) error {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	_ = listener.Close()
	return nil
}

func listenUdp(port int) error {
	udpAddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	_ = udpConn.Close()
	return nil
}
