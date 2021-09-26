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

package traversal

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mysteriumnetwork/node/core/port"
)

const portCount = 10

func TestPinger_PingPeer_N_Connections(t *testing.T) {
	pingConfig := &PingConfig{
		Interval:            5 * time.Millisecond,
		SendConnACKInterval: 5 * time.Millisecond,
		Timeout:             5 * time.Second,
	}
	provider := newPinger(pingConfig)
	consumer := newPinger(pingConfig)
	var pPorts, cPorts []int
	ports, err := port.NewFixedRangePool(port.Range{Start: 10000, End: 60000}).AcquireMultiple(portCount * 2)
	assert.NoError(t, err)
	for i := 0; i < portCount; i++ {
		pPorts = append(pPorts, ports[i].Num())
		cPorts = append(cPorts, ports[portCount+i].Num())
	}
	peerConns := make(chan *net.UDPConn, 2)
	go func() {
		conns, err := consumer.PingProviderPeer(context.Background(), "", "127.0.0.1", cPorts, pPorts, 128, 2)
		require.NoError(t, err)
		require.Len(t, conns, 2)
		peerConns <- conns[0]
		peerConns <- conns[1]
	}()
	conns, err := provider.PingConsumerPeer(context.Background(), "id", "127.0.0.1", pPorts, cPorts, 2, 2)
	if err != nil {
		t.Errorf("PingConsumerPeer error: %v", err)
		return
	}

	if len(conns) != 2 {
		t.Error("Not enough connections received from pinger")
		return
	}
	conn1 := conns[0]
	conn2 := conns[1]
	peerConn1 := <-peerConns
	peerConn2 := <-peerConns
	conn1.Close()
	conn2.Close()
	peerConn1.Close()
	peerConn2.Close()
	assert.Equal(t, conn1.RemoteAddr().(*net.UDPAddr).Port, peerConn1.LocalAddr().(*net.UDPAddr).Port)
	assert.Equal(t, conn2.RemoteAddr().(*net.UDPAddr).Port, peerConn2.LocalAddr().(*net.UDPAddr).Port)
}

func TestPinger_PingPeer_Not_Enough_Connections_Timeout(t *testing.T) {
	pingConfig := &PingConfig{
		Interval: 10 * time.Millisecond,
		Timeout:  300 * time.Millisecond,
	}

	provider := newPinger(pingConfig)
	consumer := newPinger(pingConfig)

	var pPorts, cPorts []int
	ports, err := port.NewFixedRangePool(port.Range{Start: 10000, End: 60000}).AcquireMultiple(10)
	assert.NoError(t, err)

	for i := 0; i < 5; i++ {
		pPorts = append(pPorts, ports[i].Num())
		cPorts = append(cPorts, ports[5+i].Num())
	}

	consumerPingErr := make(chan error)
	go func() {
		_, err := consumer.PingProviderPeer(context.Background(), "", "127.0.0.1", cPorts, pPorts, 2, 30)
		consumerPingErr <- err
	}()
	conns, err := provider.PingConsumerPeer(context.Background(), "id", "127.0.0.1", pPorts, cPorts, 2, 30)
	assert.Equal(t, ErrTooFew, err)
	assert.Len(t, conns, 0)

	consumerErr := <-consumerPingErr
	assert.Equal(t, ErrTooFew, consumerErr)
}

func TestPinger_PingConsumerPeer_Timeout(t *testing.T) {
	pinger := newPinger(&PingConfig{
		Interval: 1 * time.Millisecond,
		Timeout:  5 * time.Millisecond,
	})
	ports, err := port.NewFixedRangePool(port.Range{Start: 10000, End: 60000}).AcquireMultiple(10)
	assert.NoError(t, err)

	providerPort := ports[0].Num()
	consumerPort := ports[1].Num()

	go func() {
		addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("127.0.0.1:%d", providerPort))
		conn, err := net.ListenUDP("udp4", addr)
		assert.NoError(t, err)
		defer conn.Close()

		select {}
	}()

	_, err = pinger.PingConsumerPeer(context.Background(), "id", "127.0.0.1", []int{consumerPort}, []int{providerPort}, 2, 2)

	assert.Equal(t, ErrTooFew, err)
}

func newPinger(config *PingConfig) NATPinger {
	return NewPinger(config, &mockPublisher{})
}

type mockPublisher struct{}

func (p mockPublisher) Publish(topic string, data interface{}) {
}
