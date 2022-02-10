/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/services/wireguard"
	wireguard_connection "github.com/mysteriumnetwork/node/services/wireguard/connection"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint/userspace"
	"github.com/mysteriumnetwork/node/services/wireguard/key"
	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
)

const (
	// Taken from android-wireguard project
	androidTunMtu = 1280
)

// WireguardTunnelSetup exposes api for caller to implement external tunnel setup
type WireguardTunnelSetup interface {
	NewTunnel()
	AddTunnelAddress(ip string, prefixLen int)
	AddRoute(route string, prefixLen int)
	AddDNS(ip string)
	SetBlocking(blocking bool)
	Establish() (int, error)
	SetMTU(mtu int)
	Protect(socket int) error
	SetSessionName(session string)
}

type wireGuardOptions struct {
	statsUpdateInterval time.Duration
	handshakeTimeout    time.Duration
}

// NewWireGuardConnection creates a new wireguard connection
func NewWireGuardConnection(opts wireGuardOptions, device wireguardDevice, ipResolver ip.Resolver, handshakeWaiter wireguard_connection.HandshakeWaiter) (connection.Connection, error) {
	privateKey, err := key.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}

	return &wireguardConnection{
		done:            make(chan struct{}),
		stateCh:         make(chan connectionstate.State, 100),
		opts:            opts,
		device:          device,
		privateKey:      privateKey,
		ipResolver:      ipResolver,
		handshakeWaiter: handshakeWaiter,
	}, nil
}

type wireguardConnection struct {
	ports           []int
	closeOnce       sync.Once
	done            chan struct{}
	stateCh         chan connectionstate.State
	opts            wireGuardOptions
	privateKey      string
	device          wireguardDevice
	ipResolver      ip.Resolver
	handshakeWaiter wireguard_connection.HandshakeWaiter
}

var _ connection.Connection = &wireguardConnection{}

func (c *wireguardConnection) State() <-chan connectionstate.State {
	return c.stateCh
}

func (c *wireguardConnection) Statistics() (connectionstate.Statistics, error) {
	stats, err := c.device.Stats()
	if err != nil {
		return connectionstate.Statistics{}, err
	}
	return connectionstate.Statistics{
		At:            time.Now(),
		BytesSent:     stats.BytesSent,
		BytesReceived: stats.BytesReceived,
	}, nil
}

func (c *wireguardConnection) Reconnect(ctx context.Context, options connection.ConnectOptions) (err error) {
	return c.Start(ctx, options)
}

func (c *wireguardConnection) Start(ctx context.Context, options connection.ConnectOptions) (err error) {
	var config wireguard.ServiceConfig
	err = json.Unmarshal(options.SessionConfig, &config)
	if err != nil {
		return errors.Wrap(err, "could not parse wireguard session config")
	}

	c.stateCh <- connectionstate.Connecting

	defer func() {
		if err != nil {
			c.Stop()
		}
	}()

	if options.ProviderNATConn != nil {
		options.ProviderNATConn.Close()
		config.LocalPort = options.ProviderNATConn.LocalAddr().(*net.UDPAddr).Port
		config.Provider.Endpoint.Port = options.ProviderNATConn.RemoteAddr().(*net.UDPAddr).Port
	}

	if err = c.device.Start(c.privateKey, config, options.ChannelConn, options.Params.DNS); err != nil {
		return errors.Wrap(err, "could not start device")
	}

	if err = c.handshakeWaiter.Wait(ctx, c.device.Stats, c.opts.handshakeTimeout, c.done); err != nil {
		return errors.Wrap(err, "failed to handshake")
	}

	log.Debug().Msg("Connected successfully")
	c.stateCh <- connectionstate.Connected
	return nil
}

func (c *wireguardConnection) Stop() {
	c.closeOnce.Do(func() {
		c.stateCh <- connectionstate.Disconnecting
		c.device.Stop()
		c.stateCh <- connectionstate.NotConnected

		close(c.stateCh)
		close(c.done)
	})
}

func (c *wireguardConnection) GetConfig() (connection.ConsumerConfig, error) {
	if c.privateKey == "" {
		return nil, errors.New("private key is missing")
	}
	publicKey, err := key.PrivateKeyToPublicKey(c.privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not get public key from private key")
	}

	return wireguard.ConsumerConfig{
		PublicKey: publicKey,
		Ports:     c.ports,
	}, nil
}

type wireguardDevice interface {
	Start(privateKey string, config wireguard.ServiceConfig, channelConn *net.UDPConn, dns connection.DNSOption) error
	Stop()
	Stats() (wgcfg.Stats, error)
}

func newWireguardDevice(tunnelSetup WireguardTunnelSetup) wireguardDevice {
	return &wireguardDeviceImpl{tunnelSetup: tunnelSetup}
}

type wireguardDeviceImpl struct {
	tunnelSetup WireguardTunnelSetup

	device *device.Device
}

func (w *wireguardDeviceImpl) Start(privateKey string, config wireguard.ServiceConfig, channelConn *net.UDPConn, dns connection.DNSOption) error {
	log.Debug().Msg("Creating tunnel device")
	tunDevice, err := w.newTunnDevice(w.tunnelSetup, config, dns)
	if err != nil {
		return errors.Wrap(err, "could not create tunnel device")
	}

	oldDevice := w.device
	defer func() {
		if oldDevice != nil {
			oldDevice.Close()
		}
	}()

	w.device = device.NewDevice(tunDevice, conn.NewDefaultBind(), device.NewLogger(device.LogLevelVerbose, "[userspace-wg]"))

	err = w.applyConfig(w.device, privateKey, config)
	if err != nil {
		return errors.Wrap(err, "could not setup device configuration")
	}
	w.device.Up()
	socket, err := peekLookAtSocketFd4(w.device)
	if err != nil {
		return errors.Wrap(err, "could not get socket")
	}
	err = w.tunnelSetup.Protect(socket)
	if err != nil {
		return errors.Wrap(err, "could not protect socket")
	}

	// Exclude p2p channel traffic from VPN tunnel.
	if channelConn != nil {
		channelSocket, err := peekLookAtSocketFd4From(channelConn)
		if err != nil {
			return fmt.Errorf("could not get channel socket: %w", err)
		}
		err = w.tunnelSetup.Protect(channelSocket)
		if err != nil {
			return fmt.Errorf("could not protect p2p socket: %w", err)
		}
	}

	return nil
}

func (w *wireguardDeviceImpl) Stop() {
	if w.device != nil {
		w.device.Close()
	}
}

func (w *wireguardDeviceImpl) Stats() (wgcfg.Stats, error) {
	if w.device == nil {
		return wgcfg.Stats{}, errors.New("device is not started")
	}
	deviceState, err := userspace.ParseUserspaceDevice(w.device.IpcGetOperation)
	if err != nil {
		return wgcfg.Stats{}, errors.Wrap(err, "could not parse userspace wg device state")
	}
	stats, err := userspace.ParseDevicePeerStats(deviceState)
	if err != nil {
		return wgcfg.Stats{}, errors.Wrap(err, "could not get userspace wg peer stats")
	}
	return stats, nil
}

func (w *wireguardDeviceImpl) applyConfig(devApi *device.Device, privateKey string, config wireguard.ServiceConfig) error {
	deviceConfig := wgcfg.DeviceConfig{
		PrivateKey: privateKey,
		ListenPort: config.LocalPort,
		Peer: wgcfg.Peer{
			Endpoint:               &config.Provider.Endpoint,
			PublicKey:              config.Provider.PublicKey,
			KeepAlivePeriodSeconds: 18,
			// All traffic through this peer (unfortunately 0.0.0.0/0 didn't work as it was treated as ipv6)
			AllowedIPs: []string{"0.0.0.0/1", "128.0.0.0/1"},
		},
		ReplacePeers: true,
	}

	if err := devApi.IpcSetOperation(bufio.NewReader(strings.NewReader(deviceConfig.Encode()))); err != nil {
		return fmt.Errorf("could not complete ipc operation: %w", err)
	}
	return nil
}

func (w *wireguardDeviceImpl) newTunnDevice(wgTunnSetup WireguardTunnelSetup, config wireguard.ServiceConfig, dns connection.DNSOption) (tun.Device, error) {
	consumerIP := config.Consumer.IPAddress
	prefixLen, _ := consumerIP.Mask.Size()
	wgTunnSetup.NewTunnel()
	wgTunnSetup.SetSessionName("wg-tun-session")
	wgTunnSetup.AddTunnelAddress(consumerIP.IP.String(), prefixLen)
	wgTunnSetup.SetMTU(androidTunMtu)
	wgTunnSetup.SetBlocking(true)

	dnsIPs, err := dns.ResolveIPs(config.Consumer.DNSIPs)
	if err != nil {
		return nil, err
	}
	for _, dnsIP := range dnsIPs {
		wgTunnSetup.AddDNS(dnsIP)
	}

	// Route all traffic through tunnel
	wgTunnSetup.AddRoute("0.0.0.0", 1)
	wgTunnSetup.AddRoute("128.0.0.0", 1)
	wgTunnSetup.AddRoute("::", 1)
	wgTunnSetup.AddRoute("8000::", 1)

	fd, err := wgTunnSetup.Establish()
	if err != nil {
		return nil, err
	}
	log.Info().Msgf("Tun value is: %d", fd)
	tunDevice, err := newDeviceFromFd(fd)
	if err == nil {
		// non-fatal
		name, nameErr := tunDevice.Name()
		log.Info().Err(nameErr).Msg("Name value: " + name)
	}

	return tunDevice, err
}
