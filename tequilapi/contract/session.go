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
	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/mysteriumnetwork/node/consumer/session"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// NewSessionQuery creates session query with default values.
func NewSessionQuery() SessionQuery {
	return SessionQuery{}
}

// SessionQuery allows to filter requested sessions.
// swagger:parameters sessionStatsAggregated sessionStatsDaily
type SessionQuery struct {
	// Filter the sessions from this date. Formatted in RFC3339 e.g. 2020-07-01.
	// in: query
	DateFrom *strfmt.Date `json:"date_from"`

	// Filter the sessions until this date. Formatted in RFC3339 e.g. 2020-07-30.
	// in: query
	DateTo *strfmt.Date `json:"date_to"`

	// Direction to filter the sessions by. Possible values are "Provided", "Consumed".
	// in: query
	Direction *string `json:"direction"`

	// Consumer identity to filter the sessions by.
	// in: query
	ConsumerID *string `json:"consumer_id"`

	// Hermes ID to filter the sessions by.
	// in: query
	HermesID *string `json:"hermes_id"`

	// Provider identity to filter the sessions by.
	// in: query
	ProviderID *string `json:"provider_id"`

	// Service type to filter the sessions by.
	// in: query
	ServiceType *string `json:"service_type"`

	// Status to filter the sessions by. Possible values are "New", "Completed".
	// in: query
	Status *string `json:"status"`
}

// Bind creates and validates query from API request.
func (q *SessionQuery) Bind(request *http.Request) *apierror.APIError {
	v := apierror.NewValidator()

	qs := request.URL.Query()
	if qStr := qs.Get("date_from"); qStr != "" {
		if qVal, err := parseDate(qStr); err != nil {
			v.Invalid("date_from", "Cannot parse 'date_from'")
		} else {
			q.DateFrom = qVal
		}
	}
	if qStr := qs.Get("date_to"); qStr != "" {
		if qVal, err := parseDate(qStr); err != nil {
			v.Invalid("date_to", "Cannot parse 'date_to'")
		} else {
			q.DateTo = qVal
		}
	}
	if qStr := qs.Get("direction"); qStr != "" {
		q.Direction = &qStr
	}
	if qStr := qs.Get("consumer_id"); qStr != "" {
		q.ConsumerID = &qStr
	}
	if qStr := qs.Get("hermes_id"); qStr != "" {
		q.HermesID = &qStr
	}
	if qStr := qs.Get("provider_id"); qStr != "" {
		q.ProviderID = &qStr
	}
	if qStr := qs.Get("service_type"); qStr != "" {
		q.ServiceType = &qStr
	}
	if qStr := qs.Get("status"); qStr != "" {
		q.Status = &qStr
	}

	return v.Err()
}

// ToFilter converts API query to storage filter.
func (q *SessionQuery) ToFilter() *session.Filter {
	filter := session.NewFilter()
	if q.DateFrom != nil {
		filter.SetStartedFrom(time.Time(*q.DateFrom).Truncate(24 * time.Hour))
	}
	if q.DateTo != nil {
		filter.SetStartedTo(time.Time(*q.DateTo).Truncate(24 * time.Hour).Add(23 * time.Hour).Add(59 * time.Minute).Add(59 * time.Second))
	}
	if q.Direction != nil {
		filter.SetDirection(*q.Direction)
	}
	if q.ConsumerID != nil {
		filter.SetConsumerID(identity.FromAddress(*q.ConsumerID))
	}
	if q.HermesID != nil {
		filter.SetHermesID(*q.HermesID)
	}
	if q.ProviderID != nil {
		filter.SetProviderID(identity.FromAddress(*q.ProviderID))
	}
	if q.ServiceType != nil {
		filter.SetServiceType(*q.ServiceType)
	}
	if q.Status != nil {
		filter.SetStatus(*q.Status)
	}
	return filter
}

// NewSessionListQuery creates session list with default values.
func NewSessionListQuery() SessionListQuery {
	return SessionListQuery{
		PaginationQuery: NewPaginationQuery(),
	}
}

// SessionListQuery allows to filter requested sessions.
// swagger:parameters sessionList
type SessionListQuery struct {
	PaginationQuery
	SessionQuery
}

// Bind creates and validates query from API request.
func (q *SessionListQuery) Bind(request *http.Request) *apierror.APIError {
	v := apierror.NewValidator()
	if err := q.PaginationQuery.Bind(request); err != nil {
		for field, fieldErr := range err.Err.Fields {
			v.Fail(field, fieldErr.Code, fieldErr.Message)
		}
	}
	if err := q.SessionQuery.Bind(request); err != nil {
		for field, fieldErr := range err.Err.Fields {
			v.Fail(field, fieldErr.Code, fieldErr.Message)
		}
	}
	return v.Err()
}

// NewSessionListResponse maps to API session list.
func NewSessionListResponse(sessions []session.History, paginator *utils.Paginator) SessionListResponse {
	dtoArray := make([]SessionDTO, len(sessions))
	for i, se := range sessions {
		dtoArray[i] = NewSessionDTO(se)
	}

	return SessionListResponse{
		Items:       dtoArray,
		PageableDTO: NewPageableDTO(paginator),
	}
}

// SessionListResponse defines session list representable as json.
// swagger:model SessionListResponse
type SessionListResponse struct {
	Items []SessionDTO `json:"items"`
	PageableDTO
}

// NewSessionStatsAggregatedResponse maps to API aggregated stats.
func NewSessionStatsAggregatedResponse(stats session.Stats) SessionStatsAggregatedResponse {
	return SessionStatsAggregatedResponse{
		Stats: NewSessionStatsDTO(stats),
	}
}

// SessionStatsAggregatedResponse defines aggregated sessions stats response as json.
// swagger:model SessionStatsAggregatedResponse
type SessionStatsAggregatedResponse struct {
	Stats SessionStatsDTO `json:"stats"`
}

// NewSessionStatsDailyResponse maps to API session stats grouped by day.
func NewSessionStatsDailyResponse(stats session.Stats, statsDaily map[time.Time]session.Stats) SessionStatsDailyResponse {
	dtoMap := make(map[string]SessionStatsDTO, len(statsDaily))
	for date, stats := range statsDaily {
		dtoMap[date.Format("2006-01-02")] = NewSessionStatsDTO(stats)
	}

	return SessionStatsDailyResponse{
		Items: dtoMap,
		Stats: NewSessionStatsDTO(stats),
	}
}

// SessionStatsDailyResponse defines session stats representable as json.
// swagger:model SessionStatsDailyResponse
type SessionStatsDailyResponse struct {
	Items map[string]SessionStatsDTO `json:"items"`
	Stats SessionStatsDTO            `json:"stats"`
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
		IPType:          se.IPType,
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

	// example: residential
	IPType string `json:"ip_type"`
}
