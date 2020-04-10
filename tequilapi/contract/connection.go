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

package contract

import (
	"github.com/mysteriumnetwork/node/consumer/bandwidth"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/payments/crypto"
)

// NewConnectionStatisticsDTO maps to API connection stats.
func NewConnectionStatisticsDTO(connection connection.Status, statistics connection.Statistics, throughput bandwidth.Throughput, invoice crypto.Invoice) ConnectionStatisticsDTO {
	return ConnectionStatisticsDTO{
		Duration:           int(connection.Duration().Seconds()),
		BytesSent:          statistics.BytesSent,
		BytesReceived:      statistics.BytesReceived,
		ThroughputSent:     datasize.BitSize(throughput.Up).Bits(),
		ThroughputReceived: datasize.BitSize(throughput.Down).Bits(),
		TokensSpent:        invoice.AgreementTotal,
	}
}

// ConnectionStatisticsDTO holds consumer connection details.
// swagger:model ConnectionStatisticsDTO
type ConnectionStatisticsDTO struct {
	// example: 1024
	BytesSent uint64 `json:"bytes_sent"`

	// example: 1024
	BytesReceived uint64 `json:"bytes_received"`

	// Upload speed in bits per second
	// example: 1024
	ThroughputSent uint64 `json:"throughput_sent"`

	// Download speed in bits per second
	// example: 1024
	ThroughputReceived uint64 `json:"throughput_received"`

	// connection duration in seconds
	// example: 60
	Duration int `json:"duration"`

	// example: 500000
	TokensSpent uint64 `json:"tokens_spent"`
}
