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
	"net"
	"strings"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/key"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
)

const (
	//taken from android-wireguard project
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
}

// Create creates a new wireguard connection
func (wcf *WireguardConnectionFactory) Create(stateChannel connection.StateChannel, statisticsChannel connection.StatisticsChannel) (connection.Connection, error) {
	privateKey, err := key.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}

	deviceFactory := func(options connection.ConnectOptions) (*device.Device, error) {
		var config wireguard.ServiceConfig
		err := json.Unmarshal(options.SessionConfig, &config)
		if err != nil {
			return nil, err
		}

		config.Consumer.PrivateKey = privateKey

		wcf.tunnelSetup.NewTunnel()
		wcf.tunnelSetup.SetSessionName("wg-tun-session")
		//TODO fetch from user connection options
		wcf.tunnelSetup.AddDNS("8.8.8.8")

		//TODO this heavy linfting might go to doInit
		tunDevice, err := newTunnDevice(wcf.tunnelSetup, &config)
		if err != nil {
			return nil, err
		}

		devApi := device.NewDevice(tunDevice, device.NewLogger(device.LogLevelDebug, "[userspace-wg]"))
		err = setupWireguardDevice(devApi, &config)
		if err != nil {
			devApi.Close()
			return nil, err
		}
		devApi.Up()
		socket, err := peekLookAtSocketFd4(devApi)
		if err != nil {
			devApi.Close()
			return nil, err
		}
		err = wcf.tunnelSetup.Protect(socket)
		if err != nil {
			devApi.Close()
			return nil, err
		}
		return devApi, nil
	}

	return &wireguardConnection{
		deviceFactory:     deviceFactory,
		privKey:           privateKey,
		stopChannel:       make(chan struct{}),
		stateChannel:      stateChannel,
		statisticsChannel: statisticsChannel,
		stopCompleted:     &sync.WaitGroup{},
	}, nil
}

type deviceFactory func(options connection.ConnectOptions) (*device.Device, error)

func setupWireguardDevice(devApi *device.Device, config *wireguard.ServiceConfig) error {
	deviceConfig := wireguard.DeviceConfig{
		PrivateKey: config.Consumer.PrivateKey,
		ListenPort: 0,
	}

	if err := devApi.IpcSetOperation(bufio.NewReader(strings.NewReader(deviceConfig.Encode()))); err != nil {
		return err
	}

	peer := wireguard.Peer{
		PublicKey:       config.Provider.PublicKey,
		Endpoint:        &config.Provider.Endpoint,
		KeepAlivePeriod: 20,
		//all traffic through this peer (unfortunately 0.0.0.0/0 didn't work as it was treated as ipv6)
		AllowedIPs: []string{"0.0.0.0/1", "128.0.0.0/1"},
	}
	if err := devApi.IpcSetOperation(bufio.NewReader(strings.NewReader(peer.Encode()))); err != nil {
		return err
	}
	return nil
}

func newTunnDevice(wgTunnSetup WireguardTunnelSetup, config *wireguard.ServiceConfig) (tun.Device, error) {
	consumerIP := config.Consumer.IPAddress
	prefixLen, _ := consumerIP.Mask.Size()
	wgTunnSetup.AddTunnelAddress(consumerIP.IP.String(), prefixLen)
	wgTunnSetup.SetMTU(androidTunMtu)
	wgTunnSetup.SetBlocking(true)

	//route all traffic through tunnel
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
	tun, err := newDeviceFromFd(fd)
	if err == nil {
		//non-fatal
		name, nameErr := tun.Name()
		log.Info().Err(nameErr).Msg("Name value: " + name)
	}

	return tun, err
}

type wireguardConnection struct {
	privKey           string
	deviceFactory     deviceFactory
	device            *device.Device
	stopChannel       chan struct{}
	stateChannel      connection.StateChannel
	statisticsChannel connection.StatisticsChannel
	stopCompleted     *sync.WaitGroup
}

func (wg *wireguardConnection) Start(options connection.ConnectOptions) error {
	log.Debug().Msg("Creating device")
	device, err := wg.deviceFactory(options)
	if err != nil {
		return errors.Wrap(err, "failed to start wireguard connection")
	}

	wg.device = device
	wg.stateChannel <- connection.Connecting

	if err := wg.doInit(); err != nil {
		return errors.Wrap(err, "failed to start wireguard connection")
	}

	log.Debug().Msg("Emitting connected event")
	wg.stateChannel <- connection.Connected
	return nil
}

func (wg *wireguardConnection) doInit() error {
	log.Debug().Msg("Starting doInit()")
	wg.stopCompleted.Add(1)
	go wg.runPeriodically(time.Second)

	return wg.waitHandshake()
}

func (wg *wireguardConnection) Wait() error {
	wg.stopCompleted.Wait()
	return nil
}

func (wg *wireguardConnection) Stop() {
	wg.stateChannel <- connection.Disconnecting
	wg.updateStatistics()
	close(wg.stopChannel)
}

func (wg *wireguardConnection) GetConfig() (connection.ConsumerConfig, error) {
	publicKey, err := key.PrivateKeyToPublicKey(wg.privKey)
	if err != nil {
		return nil, err
	}
	return wireguard.ConsumerConfig{
		PublicKey: publicKey,
	}, nil
}

var _ connection.Connection = &wireguardConnection{}

func (wg *wireguardConnection) updateStatistics() {
	stats, err := wg.getDeviceStats()
	if err != nil {
		log.Error().Err(err).Msg("Error updating statistics")
		return
	}

	wg.statisticsChannel <- consumer.SessionStatistics{
		BytesSent:     stats.BytesSent,
		BytesReceived: stats.BytesReceived,
	}
}

func (wg *wireguardConnection) getDeviceStats() (*wireguard.Stats, error) {
	deviceState, err := wireguard.ParseUserspaceDevice(wg.device.IpcGetOperation)
	if err != nil {
		return nil, err
	}
	stats, err := wireguard.ParseDevicePeerStats(deviceState)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (wg *wireguardConnection) doCleanup() {
	wg.device.Close()
	wg.device.Wait()
	wg.stateChannel <- connection.NotConnected
	close(wg.stateChannel)
	wg.stopCompleted.Done()
}

func (wg *wireguardConnection) runPeriodically(duration time.Duration) {
	for {
		select {
		case <-time.After(duration):
			wg.updateStatistics()

		case <-wg.stopChannel:
			wg.doCleanup()
			return
		}
	}
}

func (wg *wireguardConnection) waitHandshake() error {
	// We need to send any packet to initialize handshake process
	_, _ = net.DialTimeout("tcp", "8.8.8.8:53", 100*time.Millisecond)
	for {
		select {
		case <-time.After(20 * time.Millisecond):
			stats, err := wg.getDeviceStats()
			if err != nil {
				return err
			}
			if !stats.LastHandshake.IsZero() {
				return nil
			}
		case <-wg.stopChannel:
			wg.doCleanup()
			return errors.New("stop received")
		}
	}
}
