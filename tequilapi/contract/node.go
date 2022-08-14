/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

	"github.com/shopspring/decimal"

	"github.com/mysteriumnetwork/node/core/node"
)

// NodeStatusResponse a node status reflects monitoring agent POV on node availability
// swagger:model NodeStatusResponse
type NodeStatusResponse struct {
	Status node.MonitoringStatus `json:"status"`
}

// MonitoringAgentResponse reflects amount of connectivity statuses for each service_type.
// swagger:model MonitoringAgentResponse
type MonitoringAgentResponse struct {
	Statuses node.MonitoringAgentStatuses `json:"statuses"`
	Error    string                       `json:"error,omitempty"`
}

// ProviderSessionsResponse reflects a list of sessions metrics during a period of time.
// swagger:model ProviderSessionsResponse
type ProviderSessionsResponse struct {
	Sessions []ProviderSession `json:"sessions"`
}

// NewProviderSessionsResponse creates response from node.SessionItem slice
func NewProviderSessionsResponse(sessionItems []node.SessionItem) *ProviderSessionsResponse {
	var r = ProviderSessionsResponse{}
	for _, si := range sessionItems {
		earningsDecimalEther, err := decimal.NewFromString(si.Earning)
		if err != nil {
			earningsDecimalEther = decimal.Zero
		}

		r.Sessions = append(r.Sessions, ProviderSession{
			ID:               si.ID,
			ConsumerCountry:  si.ConsumerCountry,
			ServiceType:      si.ServiceType,
			DurationSeconds:  si.Duration,
			StartedAt:        time.Unix(si.StartedAt, 0).Format(time.RFC3339),
			Earnings:         NewTokensFromDecimal(earningsDecimalEther),
			TransferredBytes: si.Transferred,
		})
	}
	return &r
}

// ProviderSessionsCountResponse reflects a number of sessions during a period of time.
// swagger:model ProviderSessionsCountResponse
type ProviderSessionsCountResponse struct {
	Count int `json:"count"`
}

// ProviderSession contains provided session ifnromation
// swagger:model ProviderSession
type ProviderSession struct {
	ID               string `json:"id"`
	ConsumerCountry  string `json:"consumer_country"`
	ServiceType      string `json:"service_type"`
	DurationSeconds  int64  `json:"duration_seconds"`
	StartedAt        string `json:"started_at"`
	Earnings         Tokens `json:"earnings"`
	TransferredBytes int64  `json:"transferred_bytes"`
}
