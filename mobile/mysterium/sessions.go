/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package mysterium

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/mysteriumnetwork/node/consumer/session"
)

// SessionStorage provides access to session history
type SessionStorage interface {
	List(*session.Filter) ([]session.History, error)
}

// SessionFilter allows to filter by time slice
/*
* arguments will be attempted to be parsed by time.RFC3339
*
* @see https://golang.org/pkg/time/
*
* @param StartedFrom - e.g. "2006-01-02T15:04:05Z" or null if undetermined
* @param StartedTo - e.g. "2006-04-02T15:04:05Z" or null if undetermined
 */
type SessionFilter struct {
	StartedFrom string
	StartedTo   string
}

// ListConsumerSessions list consumer sessions
func (mb *MobileNode) ListConsumerSessions(filter *SessionFilter) ([]byte, error) {
	d := session.DirectionConsumed

	f := &session.Filter{
		Direction: &d,
	}

	if len(filter.StartedFrom) > 0 {
		from, err := time.Parse(time.RFC3339, filter.StartedFrom)
		if err != nil {
			return nil, err
		}
		f.StartedFrom = &from
	}

	if len(filter.StartedTo) > 0 {
		to, err := time.Parse(time.RFC3339, filter.StartedTo)
		if err != nil {
			return nil, err
		}
		f.StartedTo = &to
	}

	sessions, err := mb.sessionStorage.List(f)
	if err != nil {
		return nil, err
	}

	dtos := make([]SessionDTO, len(sessions))
	for i, s := range sessions {
		dtos[i] = sessionDTO(s)
	}
	return json.Marshal(dtos)
}

// SessionDTO mobile session dto
type SessionDTO struct {
	ID              string   `json:"id"`
	Direction       string   `json:"direction"`
	ConsumerID      string   `json:"consumer_id"`
	HermesID        string   `json:"hermes_id"`
	ProviderID      string   `json:"provider_id"`
	ServiceType     string   `json:"service_type"`
	ConsumerCountry string   `json:"consumer_country"`
	ProviderCountry string   `json:"provider_country"`
	CreatedAt       string   `json:"created_at"`
	Duration        uint64   `json:"duration"`
	BytesReceived   uint64   `json:"bytes_received"`
	BytesSent       uint64   `json:"bytes_sent"`
	Tokens          *big.Int `json:"tokens"`
	Status          string   `json:"status"`
	IPType          string   `json:"ip_type"`
}

func sessionDTO(se session.History) SessionDTO {
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
