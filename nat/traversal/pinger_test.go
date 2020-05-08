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

	"github.com/mysteriumnetwork/node/core/port"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	localhostIP = net.ParseIP("127.0.0.1")
)

func TestPinger_Multiple_Stop(t *testing.T) {
	pinger := newPinger(&PingConfig{
		Interval: 1 * time.Millisecond,
		Timeout:  10 * time.Millisecond,
	})

	// Make sure multiple stops doesn't crash.
	pinger.Stop()
	pinger.Stop()
	pinger.Stop()
}

func TestPinger_Provider_Consumer_Ping_Flow(t *testing.T) {
	ports, err := port.NewPool().AcquireMultiple(4)
	assert.NoError(t, err)
	providerProxyPort := ports[0].Num()
	providerPort := ports[1].Num()
	consumerPort := ports[2].Num()
	consumerProxyPort := ports[3].Num()

	pingConfig := &PingConfig{
		Interval: 10 * time.Millisecond,
		Timeout:  2 * time.Second,
	}
	providerProxyCh := make(chan string)
	waitProxyCh := make(chan struct{})

	// Create provider's UDP proxy listener to which pinger should hand off connection.
	// In real world this proxy represents started VPN service (WireGuard or OpenVPN).
	go func() {
		conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: localhostIP, Port: providerProxyPort})
		require.NoError(t, err)
		waitProxyCh <- struct{}{}
		proxyBuf := make([]byte, 1024)
		for {
			n, err := conn.Read(proxyBuf)
			require.NoError(t, err)
			providerProxyCh <- string(proxyBuf[:n])
		}
	}()

	<-waitProxyCh

	// Start pinging consumer
	providerPinger := newPinger(pingConfig)
	defer providerPinger.Stop()
	go func() {
		providerPinger.BindServicePort("wg1", providerProxyPort)
		providerPinger.PingConsumer(context.Background(), "127.0.0.1", []int{providerPort}, []int{consumerPort}, "wg1")
	}()

	// Start pinging provider
	consumerPinger := newPinger(pingConfig)
	defer consumerPinger.Stop()
	consumerPinger.SetProtectSocketCallback(func(socket int) bool {
		return true
	})
	// Wait some time to simulate real network delay conditions.
	time.Sleep(3 * pingConfig.Interval)
	_, _, err = consumerPinger.PingProvider(context.Background(), "127.0.0.1", []int{consumerPort}, []int{providerPort}, consumerProxyPort)
	assert.NoError(t, err)

	// Create consumer conn which is used to write to consumer proxy. In real case this conn represents OpenVPN conn.
	consumerConn, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: localhostIP, Port: consumerProxyPort})
	assert.NoError(t, err)
	defer consumerConn.Close()

	assert.Eventually(t, func() bool {
		consumerConn.Write([]byte("Test message"))
		select {
		case msg := <-providerProxyCh:
			if msg == "Test message" {
				return true
			}
		default:
		}
		return false
	}, 2*time.Second, 50*time.Millisecond)
}

func TestPinger_PingPeer_N_Connections(t *testing.T) {
	pingConfig := &PingConfig{
		Interval:            50 * time.Millisecond,
		SendConnACKInterval: 50 * time.Millisecond,
		Timeout:             5 * time.Second,
	}
	provider := newPinger(pingConfig)
	consumer := newPinger(pingConfig)
	var pPorts, cPorts []int
	ports, err := port.NewPool().AcquireMultiple(20)
	assert.NoError(t, err)
	for i := 0; i < 10; i++ {
		pPorts = append(pPorts, ports[i].Num())
		cPorts = append(cPorts, ports[10+i].Num())
	}
	peerConns := make(chan *net.UDPConn, 2)
	go func() {
		conns, err := consumer.PingProviderPeer(context.Background(), "127.0.0.1", cPorts, pPorts, 128, 2)
		require.NoError(t, err)
		require.Len(t, conns, 2)
		peerConns <- conns[0]
		peerConns <- conns[1]
	}()
	conns, err := provider.PingConsumerPeer(context.Background(), "127.0.0.1", pPorts, cPorts, 2, 2)
	assert.NoError(t, err)

	assert.Len(t, conns, 2)
	conn1 := conns[0]
	conn2 := conns[1]
	peerConn1 := <-peerConns
	peerConn2 := <-peerConns
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
	ports, err := port.NewPool().AcquireMultiple(10)
	assert.NoError(t, err)

	for i := 0; i < 5; i++ {
		pPorts = append(pPorts, ports[i].Num())
		cPorts = append(cPorts, ports[5+i].Num())
	}

	consumerPingErr := make(chan error)
	go func() {
		_, err := consumer.PingProviderPeer(context.Background(), "127.0.0.1", cPorts, pPorts, 2, 30)
		consumerPingErr <- err
	}()
	conns, err := provider.PingConsumerPeer(context.Background(), "127.0.0.1", pPorts, cPorts, 2, 30)
	assert.EqualError(t, err, "ping failed: context deadline exceeded")
	assert.Len(t, conns, 0)

	consumerErr := <-consumerPingErr
	assert.EqualError(t, consumerErr, "ping failed: context deadline exceeded")
}

func TestPinger_PingProvider_Timeout(t *testing.T) {
	pinger := newPinger(&PingConfig{
		Interval: 1 * time.Millisecond,
		Timeout:  5 * time.Millisecond,
	})

	providerPort := 51205
	consumerPort := 51206

	go func() {
		addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("127.0.0.1:%d", providerPort))
		conn, err := net.ListenUDP("udp4", addr)
		assert.NoError(t, err)
		defer conn.Close()

		select {}
	}()

	_, _, err := pinger.PingProvider(context.Background(), "127.0.0.1", []int{consumerPort}, []int{providerPort}, 0)

	assert.EqualError(t, err, "failed to ping remote peer: context deadline exceeded")
}

func TestPinger_PingConsumerPeer_Timeout(t *testing.T) {
	pinger := newPinger(&PingConfig{
		Interval: 1 * time.Millisecond,
		Timeout:  5 * time.Millisecond,
	})
	ports, err := port.NewPool().AcquireMultiple(10)
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

	_, err = pinger.PingConsumerPeer(context.Background(), "127.0.0.1", []int{consumerPort}, []int{providerPort}, 2, 2)

	assert.EqualError(t, err, "ping failed: context deadline exceeded")
}

func newPinger(config *PingConfig) NATPinger {
	return NewPinger(config, &mockPublisher{})
}

type mockPublisher struct {
}

func (p mockPublisher) Publish(topic string, data interface{}) {
}
