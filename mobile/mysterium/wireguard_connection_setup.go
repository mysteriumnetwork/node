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
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/mysteriumnetwork/node/services/wireguard"
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

// WireguardConnectionFactory is the connection factory for wireguard
type WireguardConnectionFactory struct {
	tunnelSetup WireguardTunnelSetup
	ipResolver  ip.Resolver
	natPinger   natPinger
}

// Create creates a new wireguard connection
func (wcf *WireguardConnectionFactory) Create(stateChannel connection.StateChannel, statisticsChannel connection.StatisticsChannel) (connection.Connection, error) {
	privateKey, err := key.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}

	return &wireguardConnection{
		statsCheckerStop:  make(chan struct{}),
		done:              make(chan struct{}),
		pingerStop:        make(chan struct{}),
		privateKey:        privateKey,
		tunnelSetup:       wcf.tunnelSetup,
		stateChannel:      stateChannel,
		statisticsChannel: statisticsChannel,
		ipResolver:        wcf.ipResolver,
		natPinger:         wcf.natPinger,
	}, nil
}

func setupWireguardDevice(devApi *device.Device, privateKey string, config wireguard.ServiceConfig) error {
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

func newTunnDevice(wgTunnSetup WireguardTunnelSetup, config wireguard.ServiceConfig) (tun.Device, error) {
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

	// Wait for local port to become available since it will be used as WireGuard listen port
	// when provider is behind NAT.
	if config.LocalPort > 0 {
		log.Info().Msgf("Waiting for port %d to become available", config.LocalPort)
		if err := waitUDPPortReadyFor(config.LocalPort, 10*time.Second); err != nil {
			return nil, errors.Wrap(err, "failed to wait for UDP port")
		}
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

func waitUDPPortReadyFor(port int, timeout time.Duration) error {
	timeoutChan := time.After(timeout)
	for {
		select {
		case <-time.After(500 * time.Millisecond):
			p, err := net.ListenPacket("udp", fmt.Sprintf(":%d", port))
			if err != nil {
				log.Err(err).Msgf("Port %d is in use. Trying to check again...", port)
			} else {
				p.Close()
				return nil
			}
		case <-timeoutChan:
			return fmt.Errorf("timeout waiting for UDP port %d", port)
		}
	}
}

type wireguardConnection struct {
	done       chan struct{}
	pingerStop chan struct{}

	privateKey        string
	tunnelSetup       WireguardTunnelSetup
	device            *device.Device
	statsCheckerStop  chan struct{}
	stateChannel      connection.StateChannel
	statisticsChannel connection.StatisticsChannel
	ipResolver        ip.Resolver
	natPinger         natPinger
}

// TODO:(anjmao): Rewrite error handling and cleanup. Currently cleanup assumed to work only if
// int is done correctly but if it fails in any other step user will see broken state.
// See https://github.com/mysteriumnetwork/node/issues/1499.
func (c *wireguardConnection) Start(options connection.ConnectOptions) (err error) {
	var config wireguard.ServiceConfig
	err = json.Unmarshal(options.SessionConfig, &config)
	if err != nil {
		return errors.Wrap(err, "could not parse wireguard session config")
	}

	if config.LocalPort > 0 {
		err = c.natPinger.PingProvider(
			config.Provider.Endpoint.IP.String(),
			config.RemotePort,
			config.LocalPort,
			c.pingerStop,
		)
		if err != nil {
			return errors.Wrap(err, "could not ping provider")
		}
	}

	log.Debug().Msg("Creating tunnel device")
	tunDevice, err := newTunnDevice(c.tunnelSetup, config)
	if err != nil {
		return errors.Wrap(err, "could not create tunnel device")
	}

	devApi := device.NewDevice(tunDevice, device.NewLogger(device.LogLevelDebug, "[userspace-wg]"))
	defer func() {
		if err != nil && devApi != nil {
			devApi.Close()
		}
	}()

	err = setupWireguardDevice(devApi, c.privateKey, config)
	if err != nil {
		return errors.Wrap(err, "could not setup device")
	}
	devApi.Up()
	socket, err := peekLookAtSocketFd4(devApi)
	if err != nil {
		return errors.Wrap(err, "could not get socket")
	}
	err = c.tunnelSetup.Protect(socket)
	if err != nil {
		return errors.Wrap(err, "could not protect socket")
	}

	c.device = devApi
	c.stateChannel <- connection.Connecting

	go c.updateStatsPeriodically(time.Second)

	if err := wireguard.WaitHandshake(c.getDeviceStats, c.done); err != nil {
		return errors.Wrap(err, "failed to handshake")
	}

	log.Debug().Msg("Connected successfully")
	c.stateChannel <- connection.Connected
	return nil
}

func (c *wireguardConnection) Wait() error {
	<-c.done
	return nil
}

func (c *wireguardConnection) Stop() {
	c.stateChannel <- connection.Disconnecting
	c.updateStatistics()
	if c.device != nil {
		c.device.Close()
		c.device.Wait()
	}
	c.stateChannel <- connection.NotConnected

	close(c.done)
	close(c.statsCheckerStop)
	close(c.pingerStop)
	close(c.stateChannel)
	close(c.statisticsChannel)
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

func (c *wireguardConnection) updateStatistics() {
	stats, err := c.getDeviceStats()
	if err != nil {
		log.Error().Err(err).Msg("Error updating statistics")
		return
	}

	c.statisticsChannel <- consumer.SessionStatistics{
		BytesSent:     stats.BytesSent,
		BytesReceived: stats.BytesReceived,
	}
}

func (c *wireguardConnection) getDeviceStats() (*wireguard.Stats, error) {
	deviceState, err := wireguard.ParseUserspaceDevice(c.device.IpcGetOperation)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse userspace wg device state")
	}
	stats, err := wireguard.ParseDevicePeerStats(deviceState)
	if err != nil {
		return nil, errors.Wrap(err, "could not get userspace wg peer stats")
	}
	return stats, nil
}

func (c *wireguardConnection) updateStatsPeriodically(duration time.Duration) {
	for {
		select {
		case <-time.After(duration):
			c.updateStatistics()

		case <-c.statsCheckerStop:
			return
		}
	}
}
