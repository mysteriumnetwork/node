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

package userspace

import (
	"fmt"

	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
)

// ParseDevicePeerStats parses current active consumer stats.
func ParseDevicePeerStats(d *UserspaceDevice) (wgcfg.Stats, error) {
	if len(d.Peers) != 1 {
		return wgcfg.Stats{}, fmt.Errorf("exactly 1 peer expected, got %d", len(d.Peers))
	}

	p := d.Peers[0]
	return wgcfg.Stats{
		BytesSent:     uint64(p.TransmitBytes),
		BytesReceived: uint64(p.ReceiveBytes),
		LastHandshake: p.LastHandshakeTime,
	}, nil
}
