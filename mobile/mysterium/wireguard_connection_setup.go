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
	"errors"
	"os"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/services/wireguard"

	"git.zx2c4.com/wireguard-go/device"
	"git.zx2c4.com/wireguard-go/tun"
	"github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
)

// WireguardTunnelSetup exposes api for caller to implement external tunnel setup
type WireguardTunnelSetup interface {
	NewTunnel()
	AddTunnelAddress(ip string)
	AddRoute(route string)
	AddDNS(ip string)
	SetBlocking(blocking bool)
	Establish() (int, error)
	SetMTU(mtu int)
	Protect(socket int) error
	SetSessionName(session string)
}

// OverrideWireguardConnection overrides default wireguard connection implementation to more mobile adapted one
func (mobNode *MobileNode) OverrideWireguardConnection(wgTunnelSetup WireguardTunnelSetup) {
	wireguard.Bootstrap()
	mobNode.di.ConnectionRegistry.Register("wireguard", func(options connection.ConnectOptions, stateChannel connection.StateChannel, statisticsChannel connection.StatisticsChannel) (connection.Connection, error) {
		//TODO this heavy linfting might go to doInit
		tun, err := newTunnDevice(wgTunnelSetup)
		if err != nil {
			return nil, err
		}

		devApi := device.UserspaceDeviceApi(tun)
		socket, err := devApi.GetNetworkSocket()
		if err != nil {
			tun.Close()
			return nil, err
		}
		err = wgTunnelSetup.Protect(int(socket))
		if err != nil {
			tun.Close()
			return nil, err
		}
		return &wireguardConnection{
			device:            devApi,
			stopChannel:       make(chan struct{}),
			stateChannel:      stateChannel,
			statisticsChannel: statisticsChannel,
			closed:            &sync.WaitGroup{},
		}, nil
	})
}

func newTunnDevice(wgTunnSetup WireguardTunnelSetup) (tun.TUNDevice, error) {
	wgTunnSetup.NewTunnel()
	wgTunnSetup.AddTunnelAddress("10.182.47.5/24")
	wgTunnSetup.SetMTU(1280)
	wgTunnSetup.SetBlocking(true)

	fd, err := wgTunnSetup.Establish()
	if err != nil {
		return nil, err
	}
	//from this point fd is valid android tunnel and needs to be disposed to change back network to it's original state
	file := os.NewFile(uintptr(fd), "/dev/tun")
	return tun.CreateTUNFromFile(file, 1280)
}

type wireguardConnection struct {
	device            *device.DeviceApi
	wgTunnelSetup     WireguardTunnelSetup
	stopChannel       chan struct{}
	stateChannel      connection.StateChannel
	statisticsChannel connection.StatisticsChannel
	closed            *sync.WaitGroup
	cleanup           func()
}

func (wg *wireguardConnection) Start() error {
	wg.stateChannel <- connection.Connecting
	err := wg.doInit()
	if err != nil {
		wg.stateChannel <- connection.NotConnected
		return err
	}
	wg.stateChannel <- connection.Connected
	return nil
}

func (wg *wireguardConnection) doInit() error {
	wg.closed.Add(1)
	go wg.runPeriodically(time.Second)
	return nil
}

func (wg *wireguardConnection) Wait() error {
	wg.closed.Wait()
	return nil
}

func (wg *wireguardConnection) Stop() {
	wg.stateChannel <- connection.Disconnecting
	close(wg.stopChannel)
}

var _ connection.Connection = &wireguardConnection{}

func (wg *wireguardConnection) updateStatistics() {
	var err error
	defer func() {
		if err != nil {
			seelog.Error("[wg connection] Error updating statistics: ", err)
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
	wg.closed.Done()
}

func (wg *wireguardConnection) runPeriodically(duration time.Duration) {
	for {
		select {
		case <-time.After(duration):
			wg.updateStatistics()

		case <-wg.stopChannel:
			wg.doCleanup()
			break
		}
	}
}
