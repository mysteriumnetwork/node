/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package bytescount

import (
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/client/bytescount"
	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
)

// NewSessionStatsSaver returns stats handler, which saves stats stats keeper
func NewSessionStatsSaver(statisticsChannel connection.StatisticsChannel) bytescount.SessionStatsHandler {
	return func(bc bytescount.Bytecount) error {
		statisticsChannel <- consumer.SessionStatistics{BytesSent: uint64(bc.BytesOut), BytesReceived: uint64(bc.BytesIn)}
		return nil
	}
}
