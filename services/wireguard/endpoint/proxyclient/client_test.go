/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package proxyclient

import (
	"fmt"
	"net"
	"net/netip"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"

	"github.com/mysteriumnetwork/node/services/wireguard/endpoint/netstack"
	"github.com/mysteriumnetwork/node/services/wireguard/key"
	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
)

func Test_ConfigureDevice_ConfigureErrors(t *testing.T) {

	client, err := New()
	assert.NoError(t, err)

	tests := []struct {
		name     string
		config   wgcfg.DeviceConfig
		expected string
	}{
		{
			name:     "empty config",
			config:   wgcfg.DeviceConfig{},
			expected: "could not parse local addr",
		},
		{
			name: "DNS list not provided",
			config: wgcfg.DeviceConfig{
				Subnet: net.IPNet{IP: net.ParseIP("10.0.182.2"), Mask: net.IPv4Mask(255, 255, 255, 0)},
				DNS:    []string{},
			},
			expected: "DNS addr list is empty",
		},
		{
			name: "DNS list contain empty value",
			config: wgcfg.DeviceConfig{
				Subnet: net.IPNet{IP: net.ParseIP("10.0.182.2"), Mask: net.IPv4Mask(255, 255, 255, 0)},
				DNS:    []string{""},
			},
			expected: "could not parse DNS addr",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.ErrorContains(t, client.ConfigureDevice(test.config), test.expected)
		})
	}
}

// newTestDevice creates a minimal WireGuard device for testing.
func newTestDevice(t *testing.T) *device.Device {
	t.Helper()
	localAddr := netip.MustParseAddr("10.0.0.1")
	dnsAddr := netip.MustParseAddr("8.8.8.8")
	tunnel, _, err := netstack.CreateNetTUN([]netip.Addr{localAddr}, []netip.Addr{dnsAddr}, 1280)
	require.NoError(t, err)
	logger := device.NewLogger(device.LogLevelSilent, "(test) ")
	return device.NewDevice(tunnel, conn.NewDefaultBind(), logger)
}

// newTestConfig builds a DeviceConfig with valid keys for ConfigureDevice.
func newTestConfig(t *testing.T, proxyPort int) wgcfg.DeviceConfig {
	t.Helper()
	privKey, err := key.GeneratePrivateKey()
	require.NoError(t, err)
	peerPrivKey, err := key.GeneratePrivateKey()
	require.NoError(t, err)
	peerPubKey, err := key.PrivateKeyToPublicKey(peerPrivKey)
	require.NoError(t, err)

	return wgcfg.DeviceConfig{
		IfaceName:  "test0",
		Subnet:     net.IPNet{IP: net.ParseIP("10.0.0.2"), Mask: net.IPv4Mask(255, 255, 255, 0)},
		PrivateKey: privKey,
		DNS:        []string{"8.8.8.8"},
		Peer: wgcfg.Peer{
			PublicKey:              peerPubKey,
			AllowedIPs:             []string{"0.0.0.0/0"},
			KeepAlivePeriodSeconds: 18,
		},
		ProxyPort: proxyPort,
	}
}

func Test_Close_ReleasesDeviceImmediately(t *testing.T) {
	c, err := New()
	require.NoError(t, err)

	dev := newTestDevice(t)
	c.mu.Lock()
	c.Device = dev
	c.mu.Unlock()

	err = c.Close()
	require.NoError(t, err)

	c.mu.Lock()
	assert.Nil(t, c.Device, "Device must be nil immediately after Close")
	assert.Nil(t, c.proxyClose, "proxyClose must be nil after Close")
	c.mu.Unlock()
}

func Test_Close_ProxyCloseCalledAndNilled(t *testing.T) {
	c, err := New()
	require.NoError(t, err)

	var called atomic.Bool
	c.mu.Lock()
	c.proxyClose = func() error { called.Store(true); return nil }
	c.mu.Unlock()

	err = c.Close()
	require.NoError(t, err)
	assert.True(t, called.Load(), "proxyClose must be called during Close")

	c.mu.Lock()
	assert.Nil(t, c.proxyClose, "proxyClose must be nil after Close")
	c.mu.Unlock()
}

func Test_Close_Idempotent(t *testing.T) {
	c, err := New()
	require.NoError(t, err)

	dev := newTestDevice(t)
	c.mu.Lock()
	c.Device = dev
	c.mu.Unlock()

	require.NoError(t, c.Close())
	require.NoError(t, c.Close(), "second Close must not panic or error")
}

func Test_ConfigureDevice_CleansOldResources(t *testing.T) {
	c, err := New()
	require.NoError(t, err)

	// Set up an old device and proxy closure to simulate a previous configure.
	oldDev := newTestDevice(t)
	var oldProxyClosed atomic.Bool
	c.mu.Lock()
	c.Device = oldDev
	c.proxyClose = func() error { oldProxyClosed.Store(true); return nil }
	c.mu.Unlock()

	// ConfigureDevice with a valid config — should clean old resources first.
	cfg := newTestConfig(t, 0) // port 0 = OS assigns
	err = c.ConfigureDevice(cfg)
	require.NoError(t, err)

	assert.True(t, oldProxyClosed.Load(), "old proxyClose must be called during reconfigure")

	c.mu.Lock()
	assert.NotNil(t, c.Device, "new Device must be set after ConfigureDevice")
	assert.NotNil(t, c.proxyClose, "new proxyClose must be set after ConfigureDevice")
	c.mu.Unlock()

	// Cleanup
	c.Close()
}

func Test_ConfigureDevice_DoubleConfigureSamePort(t *testing.T) {
	c, err := New()
	require.NoError(t, err)

	// First configure — picks a free port.
	cfg1 := newTestConfig(t, 0)
	err = c.ConfigureDevice(cfg1)
	require.NoError(t, err)

	c.mu.Lock()
	firstDevice := c.Device
	c.mu.Unlock()
	require.NotNil(t, firstDevice)

	// Second configure — old device/proxy must be cleaned first.
	cfg2 := newTestConfig(t, 0)
	err = c.ConfigureDevice(cfg2)
	require.NoError(t, err)

	c.mu.Lock()
	secondDevice := c.Device
	c.mu.Unlock()
	require.NotNil(t, secondDevice)
	assert.True(t, firstDevice != secondDevice, "second ConfigureDevice must create a new device")

	c.Close()
}

func Test_PeerStats_NilDeviceReturnsError(t *testing.T) {
	c, err := New()
	require.NoError(t, err)

	// Device is nil (never configured or already closed)
	_, err = c.PeerStats("any")
	assert.ErrorContains(t, err, "device is closed")
}

func Test_PeerStats_NilAfterClose(t *testing.T) {
	c, err := New()
	require.NoError(t, err)

	dev := newTestDevice(t)
	c.mu.Lock()
	c.Device = dev
	c.mu.Unlock()

	require.NoError(t, c.Close())

	_, err = c.PeerStats("any")
	assert.ErrorContains(t, err, "device is closed")
}

func Test_Proxy_SyncBindFailsOnPortConflict(t *testing.T) {
	c, err := New()
	require.NoError(t, err)

	// Occupy a port on all interfaces (same as Proxy does)
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer ln.Close()
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	// Proxy on the same port should fail synchronously
	err = c.Proxy(nil, port)
	assert.Error(t, err, "Proxy should fail when port is already bound")
	assert.Contains(t, err.Error(), "proxy listen")
}
