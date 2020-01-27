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

package mysterium

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUDPPortChecker(t *testing.T) {
	port := 51000

	udpServerStarted := make(chan struct{})
	go func() {
		addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("127.0.0.1:%d", port))
		conn, err := net.ListenUDP("udp4", addr)
		assert.NoError(t, err)
		udpServerStarted <- struct{}{}
		time.Sleep(70 * time.Millisecond)
		conn.Close()
	}()

	<-udpServerStarted

	err := waitUDPPortReadyFor(port, 200*time.Millisecond, 50*time.Millisecond)

	assert.NoError(t, err)
}
