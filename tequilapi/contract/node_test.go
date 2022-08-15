/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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
	"fmt"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/core/node"
	"github.com/stretchr/testify/assert"
)

func TestProviderSession(t *testing.T) {
	for _, data := range []struct {
		from     node.SessionItem
		expected ProviderSession
	}{
		{
			from: node.SessionItem{
				ID:              "ID",
				ConsumerCountry: "US",
				ServiceType:     "wireguard",
				Duration:        145,
				StartedAt:       1659956756,
				Earning:         "1.23456789",
				Transferred:     123456789,
			},
			expected: ProviderSession{
				ID:               "ID",
				ConsumerCountry:  "US",
				ServiceType:      "wireguard",
				DurationSeconds:  145,
				StartedAt:        time.Unix(1659956756, 0).Format(time.RFC3339),
				TransferredBytes: 123456789,
			},
		},
	} {
		t.Run(fmt.Sprintf("%+v", data), func(t *testing.T) {
			items := []node.SessionItem{data.from}
			actual := NewProviderSessionsResponse(items).Sessions[0]
			assert.Equal(t, data.expected.ID, actual.ID)
			assert.Equal(t, data.expected.ConsumerCountry, actual.ConsumerCountry)
			assert.Equal(t, data.expected.ServiceType, actual.ServiceType)
			assert.Equal(t, data.expected.DurationSeconds, actual.DurationSeconds)
			assert.Equal(t, data.expected.StartedAt, actual.StartedAt)
			assert.Equal(t, data.expected.TransferredBytes, actual.TransferredBytes)
		})
	}

}
