// +build windows

/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package userspace

import (
	"net"
	"os"

	"git.zx2c4.com/wireguard-go/device"
	"git.zx2c4.com/wireguard-go/tun"
	"github.com/pkg/errors"
	"github.com/songgao/water"
)

type nativeTun struct {
	tun    *water.Interface
	events chan tun.TUNEvent
}

// CreateTUN creates native TUN device for wireguard.
func CreateTUN(name string, subnet net.IPNet) (tun.TUNDevice, error) {
	tunDevice, err := water.New(water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			ComponentID: "tap0901",
			Network:     subnet.String(),
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new TUN device")
	}

	if err := assignIP(tunDevice.Name(), subnet); err != nil {
		return nil, errors.Wrap(err, "failed to assign IP address")
	}

	if err := renameInterface(tunDevice.Name(), name); err != nil {
		return nil, errors.Wrap(err, "failed to rename network interface")
	}

	return &nativeTun{
		tun:    tunDevice,
		events: make(chan tun.TUNEvent, 10),
	}, nil
}

func (tun *nativeTun) Name() (string, error) {
	return tun.tun.Name(), nil
}

func (tun *nativeTun) File() *os.File {
	return nil
}

func (tun *nativeTun) Events() chan tun.TUNEvent {
	return tun.events
}

func (tun *nativeTun) Read(buff []byte, offset int) (int, error) {
	return tun.tun.Read(buff[offset:])
}

func (tun *nativeTun) Write(buff []byte, offset int) (int, error) {
	return tun.tun.Write(buff[offset:])
}

func (tun *nativeTun) Close() error {
	close(tun.events)
	return tun.tun.Close()
}

func (tun *nativeTun) MTU() (int, error) {
	return device.DefaultMTU, nil
}
