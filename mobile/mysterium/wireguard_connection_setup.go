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
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/mysteriumnetwork/node/services/wireguard"
	wireguard_connection "github.com/mysteriumnetwork/node/services/wireguard/connection"
	"github.com/mysteriumnetwork/node/services/wireguard/key"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
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
func NewWireGuardConnection(opts wireGuardOptions, device wireguardDevice, ipResolver ip.Resolver, natPinger natPinger, handshakeWaiter wireguard_connection.HandshakeWaiter) (connection.Connection, error) {
	privateKey, err := key.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}

	return &wireguardConnection{
		done:            make(chan struct{}),
		pingerStop:      make(chan struct{}),
		stateCh:         make(chan connection.State, 100),
		opts:            opts,
		device:          device,
		privateKey:      privateKey,
		ipResolver:      ipResolver,
		natPinger:       natPinger,
		handshakeWaiter: handshakeWaiter,
	}, nil
}

type wireguardConnection struct {
	closeOnce       sync.Once
	done            chan struct{}
	pingerStop      chan struct{}
	stateCh         chan connection.State
	opts            wireGuardOptions
	privateKey      string
	device          wireguardDevice
	ipResolver      ip.Resolver
	natPinger       natPinger
	handshakeWaiter wireguard_connection.HandshakeWaiter
}

var _ connection.Connection = &wireguardConnection{}

func (c *wireguardConnection) State() <-chan connection.State {
	return c.stateCh
}

func (c *wireguardConnection) Statistics() (connection.Statistics, error) {
	stats, err := c.device.Stats()
	if err != nil {
		return connection.Statistics{}, err
	}
	return connection.Statistics{
		At:            time.Now(),
		BytesSent:     stats.BytesSent,
		BytesReceived: stats.BytesReceived,
	}, nil
}

func (c *wireguardConnection) Start(options connection.ConnectOptions) (err error) {
	var config wireguard.ServiceConfig
	err = json.Unmarshal(options.SessionConfig, &config)
	if err != nil {
		return errors.Wrap(err, "could not parse wireguard session config")
	}

	c.stateCh <- connection.Connecting

	defer func() {
		if err != nil {
			c.Stop()
		}
	}()

	if config.LocalPort > 0 {
		err = c.natPinger.PingProvider(
			config.Provider.Endpoint.IP.String(),
			config.RemotePort,
			config.LocalPort,
			0,
			c.pingerStop,
		)
		if err != nil {
			return errors.Wrap(err, "could not ping provider")
		}
	}

	if err := c.device.Start(c.privateKey, config); err != nil {
		return errors.Wrap(err, "could not start device")
	}

	if err := c.handshakeWaiter.Wait(c.device.Stats, c.opts.handshakeTimeout, c.done); err != nil {
		return errors.Wrap(err, "failed to handshake")
	}

	log.Debug().Msg("Connected successfully")
	c.stateCh <- connection.Connected
	return nil
}

func (c *wireguardConnection) Wait() error {
	<-c.done
	return nil
}

func (c *wireguardConnection) Stop() {
	c.closeOnce.Do(func() {
		c.stateCh <- connection.Disconnecting
		c.device.Stop()
		c.stateCh <- connection.NotConnected

		close(c.pingerStop)
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

	var publicIP string
	if !c.isNoopPinger() {
		var err error
		publicIP, err = c.ipResolver.GetPublicIP()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get consumer public IP")
		}
	}

	return wireguard.ConsumerConfig{
		PublicKey: publicKey,
		IP:        publicIP,
	}, nil
}

func (c *wireguardConnection) isNoopPinger() bool {
	_, ok := c.natPinger.(*traversal.NoopPinger)
	return ok
}

type wireguardDevice interface {
	Start(privateKey string, config wireguard.ServiceConfig) error
	Stop()
	Stats() (*wireguard.Stats, error)
}

func newWireguardDevice(tunnelSetup WireguardTunnelSetup) wireguardDevice {
	return &wireguardDeviceImpl{tunnelSetup: tunnelSetup}
}

type wireguardDeviceImpl struct {
	tunnelSetup WireguardTunnelSetup

	device *device.Device
}

func (w *wireguardDeviceImpl) Start(privateKey string, config wireguard.ServiceConfig) error {
	log.Debug().Msg("Creating tunnel device")
	tunDevice, err := w.newTunnDevice(w.tunnelSetup, config)
	if err != nil {
		return errors.Wrap(err, "could not create tunnel device")
	}

	w.device = device.NewDevice(tunDevice, device.NewLogger(device.LogLevelDebug, "[userspace-wg]"))

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
	return nil
}

func (w *wireguardDeviceImpl) Stop() {
	if w.device != nil {
		w.device.Close()
	}
}

func (w *wireguardDeviceImpl) Stats() (*wireguard.Stats, error) {
	if w.device == nil {
		return nil, errors.New("device is not started")
	}
	deviceState, err := wireguard.ParseUserspaceDevice(w.device.IpcGetOperation)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse userspace wg device state")
	}
	stats, err := wireguard.ParseDevicePeerStats(deviceState)
	if err != nil {
		return nil, errors.Wrap(err, "could not get userspace wg peer stats")
	}
	return stats, nil
}

func (w *wireguardDeviceImpl) applyConfig(devApi *device.Device, privateKey string, config wireguard.ServiceConfig) error {
	deviceConfig := wireguard.DeviceConfig{
		PrivateKey: privateKey,
		ListenPort: config.LocalPort,
	}

	if err := devApi.IpcSetOperation(bufio.NewReader(strings.NewReader(deviceConfig.Encode()))); err != nil {
		return err
	}

	peer := wireguard.Peer{
		Endpoint:               &config.Provider.Endpoint,
		PublicKey:              config.Provider.PublicKey,
		KeepAlivePeriodSeconds: 18,
		// All traffic through this peer (unfortunately 0.0.0.0/0 didn't work as it was treated as ipv6)
		AllowedIPs: []string{"0.0.0.0/1", "128.0.0.0/1"},
	}
	if err := devApi.IpcSetOperation(bufio.NewReader(strings.NewReader(peer.Encode()))); err != nil {
		return err
	}
	return nil
}

func (w *wireguardDeviceImpl) newTunnDevice(wgTunnSetup WireguardTunnelSetup, config wireguard.ServiceConfig) (tun.Device, error) {
	consumerIP := config.Consumer.IPAddress
	prefixLen, _ := consumerIP.Mask.Size()
	wgTunnSetup.NewTunnel()
	wgTunnSetup.SetSessionName("wg-tun-session")
	wgTunnSetup.AddTunnelAddress(consumerIP.IP.String(), prefixLen)
	wgTunnSetup.SetMTU(androidTunMtu)
	wgTunnSetup.SetBlocking(true)

	autoDNS := connection.DNSOptionAuto
	dnsIPs, err := autoDNS.ResolveIPs(config.Consumer.DNSIPs)
	if err != nil {
		return nil, err
	}
	for _, dnsIP := range dnsIPs {
		wgTunnSetup.AddDNS(dnsIP)
	}

	// Route all traffic through tunnel
	wgTunnSetup.AddRoute("0.0.0.0", 1)
	wgTunnSetup.AddRoute("128.0.0.0", 1)

	// Provider requests to delay consumer connection since it might be in a process of setting up NAT traversal for given consumer
	if config.Consumer.ConnectDelay > 0 {
		log.Info().Msgf("Delaying tunnel creation for %v milliseconds", config.Consumer.ConnectDelay)
		time.Sleep(time.Duration(config.Consumer.ConnectDelay) * time.Millisecond)
	}

	fd, err := wgTunnSetup.Establish()
	if err != nil {
		return nil, err
	}
	log.Info().Msgf("Tun value is: %d", fd)
	tunDevice, err := newDeviceFromFd(fd)
	if err == nil {
		//non-fatal
		name, nameErr := tunDevice.Name()
		log.Info().Err(nameErr).Msg("Name value: " + name)
	}

	return tunDevice, err
}
