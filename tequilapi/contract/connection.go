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
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/payments/crypto"
)

// NewConnectionStatisticsDTO maps consumer connection details.
func NewConnectionStatisticsDTO(connection connection.Status, statistics connection.Statistics, invoice crypto.Invoice) ConnectionStatisticsDTO {
	return ConnectionStatisticsDTO{
		Duration:      int(connection.Duration().Seconds()),
		BytesSent:     statistics.BytesSent,
		BytesReceived: statistics.BytesReceived,
		TokensSpent:   invoice.AgreementTotal,
	}
}

// ConnectionStatisticsDTO represents consumer connection details.
// swagger:model ConnectionStatisticsDTO
type ConnectionStatisticsDTO struct {
	// example: 1024
	BytesSent uint64 `json:"bytes_sent"`

	// example: 1024
	BytesReceived uint64 `json:"bytes_received"`

	// connection duration in seconds
	// example: 60
	Duration int `json:"duration"`

	// example: 500000
	TokensSpent uint64 `json:"tokens_spent"`
}
