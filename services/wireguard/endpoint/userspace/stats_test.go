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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
)

func TestParsePeerStats(t *testing.T) {
	tests := []struct {
		name          string
		device        *UserspaceDevice
		expectedStats wgcfg.Stats
		expectedErr   error
	}{
		{
			name: "Test parse stats successfully",
			device: &UserspaceDevice{
				Peers: []UserspaceDevicePeer{
					{
						TransmitBytes:     10,
						ReceiveBytes:      12,
						LastHandshakeTime: time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC),
					},
				},
			},
			expectedStats: wgcfg.Stats{
				BytesSent:     10,
				BytesReceived: 12,
				LastHandshake: time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC),
			},
			expectedErr: nil,
		},
		{
			name: "Test parse fail when more than one peer returned",
			device: &UserspaceDevice{
				Peers: []UserspaceDevicePeer{{}, {}},
			},
			expectedStats: wgcfg.Stats{},
			expectedErr:   fmt.Errorf("exactly 1 peer expected, got %d", 2),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stats, err := ParseDevicePeerStats(test.device)
			assert.Equal(t, test.expectedErr, err)
			assert.Equal(t, test.expectedStats, stats)
		})
	}
}
