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
	"time"
)

// ListServiceSessionsResponse defines session list representable as json
// swagger:model ListServiceSessionsResponse
type ListServiceSessionsResponse struct {
	Sessions []ServiceSessionDTO `json:"sessions"`
}

// ServiceSessionDTO represents the session object
// swagger:model ServiceSessionDTO
type ServiceSessionDTO struct {
	// example: 4cfb0324-daf6-4ad8-448b-e61fe0a1f918
	ID string `json:"id"`
	// example: 0x0000000000000000000000000000000000000001
	ConsumerID string `json:"consumer_id"`
	// example: 2019-06-06T11:04:43.910035Z
	CreatedAt time.Time `json:"created_at"`
	// example: 12345
	BytesOut uint64 `json:"bytes_out"`
	// example: 23451
	BytesIn uint64 `json:"bytes_in"`
	// example: 4cfb0324-daf6-4ad8-448b-e61fe0a1f918
	ServiceID string `json:"service_id"`
	// example: wireguard
	ServiceType string `json:"service_type"`
	// example: 500000
	TokensEarned uint64 `json:"tokens_earned"`
}

// ListConnectionSessionsResponse defines session list representable as json
// swagger:model ListConnectionSessionsResponse
type ListConnectionSessionsResponse struct {
	Sessions []ConnectionSessionDTO `json:"sessions"`
}

// ConnectionSessionDTO represents the session object
// swagger:model ConnectionSessionDTO
type ConnectionSessionDTO struct {
	// example: 4cfb0324-daf6-4ad8-448b-e61fe0a1f918
	SessionID string `json:"session_id"`

	// example: 0x0000000000000000000000000000000000000001
	ConsumerID string `json:"consumer_id"`

	// example: 0x0000000000000000000000000000000000000001
	HermesID string `json:"hermes_id"`

	// example: 0x0000000000000000000000000000000000000001
	ProviderID string `json:"provider_id"`

	// example: openvpn
	ServiceType string `json:"service_type"`

	// example: NL
	ProviderCountry string `json:"provider_country"`

	// example: 2018-10-29 16:22:05
	DateStarted string `json:"date_started"`

	// example: 1024
	BytesSent uint64 `json:"bytes_sent"`

	// example: 1024
	BytesReceived uint64 `json:"bytes_received"`

	// duration in seconds
	// example: 120
	Duration uint64 `json:"duration"`

	// example: 500000
	TokensSpent uint64 `json:"tokens_spent"`

	// example: Completed
	Status string `json:"status"`
}
