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

	"github.com/mysteriumnetwork/node/core/monitoring"

	"github.com/shopspring/decimal"

	"github.com/mysteriumnetwork/node/core/node"
)

// NodeStatusResponse a node status reflects monitoring agent POV on node availability
// swagger:model NodeStatusResponse
type NodeStatusResponse struct {
	Status monitoring.Status `json:"status"`
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
	r := ProviderSessionsResponse{Sessions: []ProviderSession{}}
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

// ProviderTransferredDataResponse reflects a number of bytes transferred by provider during a period of time.
// swagger:model ProviderTransferredDataResponse
type ProviderTransferredDataResponse struct {
	Bytes int `json:"transferred_data_bytes"`
}

// ProviderSessionsCountResponse reflects a number of sessions during a period of time.
// swagger:model ProviderSessionsCountResponse
type ProviderSessionsCountResponse struct {
	Count int `json:"count"`
}

// ProviderConsumersCountResponse reflects a number of unique consumers served during a period of time.
// swagger:model ProviderConsumersCountResponse
type ProviderConsumersCountResponse struct {
	Count int `json:"count"`
}

// ProviderSeriesItem represents a general data series item
type ProviderSeriesItem struct {
	Value     string `json:"value"`
	Timestamp int64  `json:"timestamp"`
}

// ProviderEarningsSeriesResponse reflects a earnings series metrics during a period of time.
// swagger:model ProviderEarningsSeriesResponse
type ProviderEarningsSeriesResponse struct {
	Data []ProviderSeriesItem `json:"data"`
}

// ProviderSessionsSeriesResponse reflects a sessions data series metrics during a period of time.
// swagger:model ProviderSessionsSeriesResponse
type ProviderSessionsSeriesResponse struct {
	Data []ProviderSeriesItem `json:"data"`
}

// ProviderTransferredDataSeriesResponse reflects a transferred bytes data series metrics during a period of time.
// swagger:model ProviderTransferredDataSeriesResponse
type ProviderTransferredDataSeriesResponse struct {
	Data []ProviderSeriesItem `json:"data"`
}

// ActivityStatsResponse reflects a node activity stats.
// swagger:model ActivityStatsResponse
type ActivityStatsResponse struct {
	Online float64 `json:"online_percent"`
	Active float64 `json:"active_percent"`
}

// QualityInfoResponse reflects a node quality.
// swagger:model QualityInfoResponse
type QualityInfoResponse struct {
	Quality float64 `json:"quality"`
}

// ProviderSession contains provided session information.
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

// LatestReleaseResponse latest release info
// swagger:model LatestReleaseResponse
type LatestReleaseResponse struct {
	Version string `json:"version"`
}

// EarningsPerServiceResponse contains information about earnings per service
// swagger:model EarningsPerServiceResponse
type EarningsPerServiceResponse struct {
	EarningsPublic        Tokens `json:"public_tokens"`
	EarningsVPN           Tokens `json:"data_transfer_tokens"`
	EarningsScraping      Tokens `json:"scraping_tokens"`
	EarningsDVPN          Tokens `json:"dvpn_tokens"`
	EarningsTotal         Tokens `json:"total_tokens"`
	TotalEarningsPublic   Tokens `json:"total_public_tokens"`
	TotalEarningsVPN      Tokens `json:"total_data_transfer_tokens"`
	TotalEarningsScraping Tokens `json:"total_scraping_tokens"`
	TotalEarningsDVPN     Tokens `json:"total_dvpn_tokens"`
}
