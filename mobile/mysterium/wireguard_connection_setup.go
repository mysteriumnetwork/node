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
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"

	"git.zx2c4.com/wireguard-go/device"
	"git.zx2c4.com/wireguard-go/tun"
	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/key"
	"github.com/pkg/errors"
)

const (
	//taken from android-wireguard project
	androidTunMtu = 1280
	tag           = "[wg connection] "
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

	deviceFactory := func(options connection.ConnectOptions) (*device.DeviceApi, error) {
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
		tun, err := newTunnDevice(wcf.tunnelSetup, &config)
		if err != nil {
			return nil, err
		}

		devApi := device.UserspaceDeviceApi(tun)
		err = setupWireguardDevice(devApi, &config)
		if err != nil {
			devApi.Close()
			return nil, err
		}
		devApi.Boot()
		socket, err := devApi.GetNetworkSocket()
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

// OverrideWireguardConnection overrides default wireguard connection implementation to more mobile adapted one
func (mobNode *MobileNode) OverrideWireguardConnection(wgTunnelSetup WireguardTunnelSetup) {
	wireguard.Bootstrap()
	factory := &WireguardConnectionFactory{
		tunnelSetup: wgTunnelSetup,
	}
	mobNode.di.ConnectionRegistry.Register(wireguard.ServiceType, factory)
}

type deviceFactory func(options connection.ConnectOptions) (*device.DeviceApi, error)

func setupWireguardDevice(devApi *device.DeviceApi, config *wireguard.ServiceConfig) error {
	err := devApi.SetListeningPort(0) //random port
	if err != nil {
		return err
	}

	privKeyArr, err := base64stringTo32ByteArray(config.Consumer.PrivateKey)
	if err != nil {
		return err
	}
	err = devApi.SetPrivateKey(device.NoisePrivateKey(privKeyArr))
	if err != nil {
		return err
	}

	peerPubKeyArr, err := base64stringTo32ByteArray(config.Provider.PublicKey)
	if err != nil {
		return err
	}

	ep := config.Provider.Endpoint.String()
	endpoint, err := device.CreateEndpoint(ep)
	if err != nil {
		return err
	}

	err = devApi.AddPeer(device.ExternalPeer{
		PublicKey:       device.NoisePublicKey(peerPubKeyArr),
		RemoteEndpoint:  endpoint,
		KeepAlivePeriod: 20,
		//all traffic through this peer (unfortunately 0.0.0.0/0 didn't work as it was treated as ipv6)
		AllowedIPs: []string{"0.0.0.0/1", "128.0.0.0/1"},
	})
	return err
}

func base64stringTo32ByteArray(s string) (res [32]byte, err error) {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if len(decoded) != 32 {
		err = errors.New("unexpected key size")
	}
	if err != nil {
		return
	}

	copy(res[:], decoded)
	return
}

func newTunnDevice(wgTunnSetup WireguardTunnelSetup, config *wireguard.ServiceConfig) (tun.TUNDevice, error) {
	consumerIP := config.Consumer.IPAddress
	prefixLen, _ := consumerIP.Mask.Size()
	wgTunnSetup.AddTunnelAddress(consumerIP.IP.String(), prefixLen)
	wgTunnSetup.SetMTU(androidTunMtu)
	wgTunnSetup.SetBlocking(true)

	//route all traffic through tunnel
	wgTunnSetup.AddRoute("0.0.0.0", 1)
	wgTunnSetup.AddRoute("128.0.0.0", 1)

	fd, err := wgTunnSetup.Establish()
	if err != nil {
		return nil, err
	}
	log.Info(tag, "Tun value is: ", fd)
	tun, err := newDeviceFromFd(fd)
	if err == nil {
		//non-fatal
		name, nameErr := tun.Name()
		log.Info(tag, "Name value: ", name, " Possible error: ", nameErr)
	}

	return tun, err
}

type wireguardConnection struct {
	privKey           string
	deviceFactory     deviceFactory
	device            *device.DeviceApi
	stopChannel       chan struct{}
	stateChannel      connection.StateChannel
	statisticsChannel connection.StatisticsChannel
	stopCompleted     *sync.WaitGroup
}

func (wg *wireguardConnection) Start(options connection.ConnectOptions) error {
	device, err := wg.deviceFactory(options)
	if err != nil {
		return errors.Wrap(err, "failed to start wireguard connection")
	}

	wg.device = device
	wg.stateChannel <- connection.Connecting

	if err := wg.doInit(); err != nil {
		return errors.Wrap(err, "failed to start wireguard connection")
	}

	wg.stateChannel <- connection.Connected
	return nil
}

func (wg *wireguardConnection) doInit() error {
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
	var err error
	defer func() {
		if err != nil {
			log.Error(tag, "Error updating statistics: ", err)
		}
	}()

	peers, err := wg.device.Peers()
	if err != nil {
		return
	}
	if len(peers) != 1 {
		err = errors.New("exactly 1 peer expected")
		return
	}
	peerStatistics := peers[0].Stats

	wg.statisticsChannel <- consumer.SessionStatistics{
		BytesSent:     peerStatistics.Sent,
		BytesReceived: peerStatistics.Received,
	}
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
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			peers, err := wg.device.Peers()
			if err != nil {
				return errors.Wrap(err, "failed to wait peer handshake")
			}
			if len(peers) != 1 {
				return errors.Wrap(errors.New("exactly 1 peer expected"), "failed to wait peer handshake")
			}
			if peers[0].LastHanshake != 0 {
				return nil
			}

		case <-wg.stopChannel:
			wg.doCleanup()
			return errors.New("stop received")
		}
	}
}
