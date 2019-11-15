// +build !windows

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

	"github.com/pkg/errors"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
)

// CreateTUN creates native TUN device for wireguard.
func CreateTUN(name string, subnet net.IPNet) (tunDevice tun.Device, err error) {
	if tunDevice, err = tun.CreateTUN(name, device.DefaultMTU); err != nil {
		return nil, errors.Wrap(err, "failed to create TUN device")
	}
	if err = assignIP(name, subnet); err != nil {
		return nil, errors.Wrap(err, "failed to assign IP address")
	}
	return tunDevice, nil
}
