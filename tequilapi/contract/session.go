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
	"math/big"
	"net/http"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/mysteriumnetwork/node/consumer/session"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/mysteriumnetwork/node/tequilapi/validation"
)

// NewSessionListQuery creates session query from API request.
func NewSessionListQuery(request *http.Request) (SessionListQuery, *validation.FieldErrorMap) {
	pagination, errs := NewPaginationQuery(request)

	query := request.URL.Query()
	return SessionListQuery{
		PaginationQuery: pagination,
		DateFrom:        parseDateOptional(query.Get("date_from"), errs.ForField("date_from")),
		DateTo:          parseDateOptional(query.Get("date_to"), errs.ForField("date_to")),
		Direction:       parseStringOptional(query.Get("direction"), errs.ForField("direction")),
		ServiceType:     parseStringOptional(query.Get("service_type"), errs.ForField("service_type")),
		Status:          parseStringOptional(query.Get("status"), errs.ForField("status")),
	}, errs
}

// SessionListQuery allows to filter requested sessions.
// swagger:parameters sessionList
type SessionListQuery struct {
	PaginationQuery

	// Filter the sessions from this date (now -30d, by default). Formatted in RFC3339 e.g. 2020-07-01.
	// in: query
	DateFrom *strfmt.Date `json:"date_from"`

	// Filter the sessions until this date (now, by default). Formatted in RFC3339 e.g. 2020-07-30.
	// in: query
	DateTo *strfmt.Date `json:"date_to"`

	// Direction to filter the sessions by. Possible values are "Provided", "Consumed".
	// in: query
	Direction *string `json:"direction"`

	// Service type to filter the sessions by.
	// in: query
	ServiceType *string `json:"service_type"`

	// Status to filter the sessions by. Possible values are "New", "Completed".
	// in: query
	Status *string `json:"status"`
}

// NewSessionListResponse maps to API session list.
func NewSessionListResponse(
	sessions []session.History,
	paginator *utils.Paginator,
	stats session.Stats,
	statsDaily map[time.Time]session.Stats,
) SessionListResponse {
	dtoArray := make([]SessionDTO, len(sessions))
	for i, se := range sessions {
		dtoArray[i] = NewSessionDTO(se)
	}

	return SessionListResponse{
		Items:       dtoArray,
		PageableDTO: NewPageableDTO(paginator),
		Stats:       NewSessionStatsDTO(stats),
		StatsDaily:  NewSessionStatsDailyDTO(statsDaily),
	}
}

// SessionListResponse defines session list representable as json.
// swagger:model SessionListResponse
type SessionListResponse struct {
	Items []SessionDTO `json:"items"`
	PageableDTO
	Stats      SessionStatsDTO            `json:"stats"`
	StatsDaily map[string]SessionStatsDTO `json:"stats_daily"`
}

// NewSessionStatsDTO maps to API session stats.
func NewSessionStatsDTO(stats session.Stats) SessionStatsDTO {
	return SessionStatsDTO{
		Count:            stats.Count,
		CountConsumers:   len(stats.ConsumerCounts),
		SumBytesReceived: stats.SumDataReceived,
		SumBytesSent:     stats.SumDataSent,
		SumDuration:      uint64(stats.SumDuration.Seconds()),
		SumTokens:        stats.SumTokens,
	}
}

// SessionStatsDTO represents the session aggregated statistics.
// swagger:model SessionStatsDTO
type SessionStatsDTO struct {
	Count            int      `json:"count"`
	CountConsumers   int      `json:"count_consumers"`
	SumBytesReceived uint64   `json:"sum_bytes_received"`
	SumBytesSent     uint64   `json:"sum_bytes_sent"`
	SumDuration      uint64   `json:"sum_duration"`
	SumTokens        *big.Int `json:"sum_tokens"`
}

// NewSessionStatsDailyDTO maps to API session stats grouped by day.
func NewSessionStatsDailyDTO(statsGrouped map[time.Time]session.Stats) map[string]SessionStatsDTO {
	dto := make(map[string]SessionStatsDTO, len(statsGrouped))
	for date, stats := range statsGrouped {
		dto[date.Format("2006-01-02")] = NewSessionStatsDTO(stats)
	}
	return dto
}

// NewSessionDTO maps to API session.
func NewSessionDTO(se session.History) SessionDTO {
	return SessionDTO{
		ID:              string(se.SessionID),
		Direction:       se.Direction,
		ConsumerID:      se.ConsumerID.Address,
		HermesID:        se.HermesID,
		ProviderID:      se.ProviderID.Address,
		ServiceType:     se.ServiceType,
		ConsumerCountry: se.ConsumerCountry,
		ProviderCountry: se.ProviderCountry,
		CreatedAt:       se.Started.Format(time.RFC3339),
		BytesReceived:   se.DataReceived,
		BytesSent:       se.DataSent,
		Duration:        uint64(se.GetDuration().Seconds()),
		Tokens:          se.Tokens,
		Status:          se.Status,
	}
}

// SessionDTO represents the session object.
// swagger:model SessionDTO
type SessionDTO struct {
	// example: 4cfb0324-daf6-4ad8-448b-e61fe0a1f918
	ID string `json:"id"`

	// example: Consumed
	Direction string `json:"direction"`

	// example: 0x0000000000000000000000000000000000000001
	ConsumerID string `json:"consumer_id"`

	// example: 0x0000000000000000000000000000000000000001
	HermesID string `json:"hermes_id"`

	// example: 0x0000000000000000000000000000000000000001
	ProviderID string `json:"provider_id"`

	// example: openvpn
	ServiceType string `json:"service_type"`

	// example: NL
	ConsumerCountry string `json:"consumer_country"`

	// example: US
	ProviderCountry string `json:"provider_country"`

	// example: 2019-06-06T11:04:43.910035Z
	CreatedAt string `json:"created_at"`

	// duration in seconds
	// example: 120
	Duration uint64 `json:"duration"`

	// example: 1024
	BytesReceived uint64 `json:"bytes_received"`

	// example: 1024
	BytesSent uint64 `json:"bytes_sent"`

	// example: 500000
	Tokens *big.Int `json:"tokens"`

	// example: Completed
	Status string `json:"status"`
}
