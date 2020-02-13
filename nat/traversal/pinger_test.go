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
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/mocks"
	"github.com/stretchr/testify/assert"
)

func TestPinger_Start_Stop(t *testing.T) {
	pinger := newPinger(&PingConfig{
		Interval: 1 * time.Millisecond,
		Timeout:  10 * time.Millisecond,
	})

	go pinger.Start()

	// Make sure multiple stops doesn't crash.
	pinger.Stop()
	pinger.Stop()
	pinger.Stop()
}

func TestPinger_Provider_Consumser_Ping_Flow(t *testing.T) {
	providerProxyPort := 51199
	providerPort := 51200
	consumerPort := 51201

	pingConfig := &PingConfig{
		Interval: 10 * time.Millisecond,
		Timeout:  100 * time.Millisecond,
	}
	pinger := newPinger(pingConfig)

	go pinger.Start()
	defer pinger.Stop()

	// Create provider's UDP proxy listener to which pinger should hand off connection.
	// In real world this proxy represents started VPN service (WireGuard or OpenVPN).
	proxyBuf := make([]byte, 1024)
	go func() {
		addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("127.0.0.1:%d", providerProxyPort))
		conn, err := net.ListenUDP("udp4", addr)
		assert.NoError(t, err)

		for {
			_, err := conn.Read(proxyBuf)
			assert.NoError(t, err)
		}
	}()

	// Start pinging consumer.
	go func() {
		pinger.BindServicePort("wg1", providerProxyPort)
		p := &Params{
			ProviderPort:        providerPort,
			ConsumerPort:        consumerPort,
			ConsumerPublicIP:    "127.0.0.1",
			ProxyPortMappingKey: "wg1",
			Cancel:              make(chan struct{}),
		}
		pinger.PingTarget(p)
	}()

	// Wait some time to simulate real network delay conditions.
	time.Sleep(5 * pingConfig.Interval)

	// Start pinging provider.
	stop := make(chan struct{})
	defer close(stop)
	err := pinger.PingProvider("127.0.0.1", providerPort, consumerPort, consumerPort+1, stop)

	assert.NoError(t, err)
	assert.Contains(t, string(proxyBuf), fmt.Sprintf("continuously pinging to 127.0.0.1:%d", providerPort))
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

	stop := make(chan struct{})
	defer close(stop)
	err := pinger.PingProvider("127.0.0.1", providerPort, consumerPort, 0, stop)
	assert.Error(t, errNATPunchAttemptTimedOut, err)
}

func newPinger(config *PingConfig) NATPinger {
	proxy := NewNATProxy()
	return NewPinger(config, proxy, mocks.NewEventBus())
}
