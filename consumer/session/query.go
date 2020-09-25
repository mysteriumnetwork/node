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

package session

import (
	"time"

	"github.com/asdine/storm/v3"
	"github.com/asdine/storm/v3/q"
)

// NewQuery creates instance of new query.
func NewQuery() *Query {
	return &Query{
		fetch: make([]q.Matcher, 0),
	}
}

// Query defines all flags for session filtering in session storage.
type Query struct {
	Sessions   []History
	Stats      Stats
	StatsByDay map[time.Time]Stats

	StartedFrom *time.Time
	StartedTo   *time.Time
	Direction   *string
	ServiceType *string
	Status      *string

	fetch []q.Matcher
}

// SetStartedFrom filters fetched sessions from given time.
func (qr *Query) SetStartedFrom(from time.Time) *Query {
	qr.StartedFrom = &from
	return qr
}

// SetStartedTo filters fetched sessions to given time.
func (qr *Query) SetStartedTo(to time.Time) *Query {
	qr.StartedTo = &to
	return qr
}

// SetDirection filters fetched sessions by direction.
func (qr *Query) SetDirection(direction string) *Query {
	qr.Direction = &direction
	return qr
}

// SetServiceType filters fetched sessions by service type.
func (qr *Query) SetServiceType(serviceType string) *Query {
	qr.ServiceType = &serviceType
	return qr
}

// SetStatus filters fetched sessions by status.
func (qr *Query) SetStatus(status string) *Query {
	qr.Status = &status
	return qr
}

// FetchSessions fetches list of sessions to Query.Sessions.
func (qr *Query) FetchSessions() *Query {
	return qr
}

// FetchStats fetches aggregated statistics to Query.Stats.
func (qr *Query) FetchStats() *Query {
	qr.Stats = NewStats()

	qr.fetch = append(
		qr.fetch,
		matcher(func(session History) bool {
			qr.Stats.Add(session)
			return true
		}),
	)

	return qr
}

const stepDay = 24 * time.Hour

// FetchStatsByDay fetches aggregated statistics grouped by day to Query.StatsByDay.
func (qr *Query) FetchStatsByDay() *Query {
	// fill the period with zeros
	qr.StatsByDay = make(map[time.Time]Stats)
	if qr.StartedFrom != nil && qr.StartedTo != nil {
		for i := qr.StartedFrom.Truncate(stepDay); !i.After(*qr.StartedTo); i = i.Add(stepDay) {
			qr.StatsByDay[i] = NewStats()
		}
	}

	qr.fetch = append(
		qr.fetch,
		matcher(func(session History) bool {
			i := session.Started.Truncate(stepDay)

			stats := qr.StatsByDay[i]
			stats.Add(session)
			qr.StatsByDay[i] = stats
			return true
		}),
	)

	return qr
}

func (qr *Query) run(node storm.Node) error {
	where := make([]q.Matcher, 0)
	if qr.StartedFrom != nil {
		where = append(where, q.Gte("Started", qr.StartedFrom))
	}
	if qr.StartedTo != nil {
		where = append(where, q.Lte("Started", qr.StartedTo))
	}
	if qr.Direction != nil {
		where = append(where, q.Eq("Direction", qr.Direction))
	}
	if qr.ServiceType != nil {
		where = append(where, q.Eq("ServiceType", qr.ServiceType))
	}
	if qr.Status != nil {
		where = append(where, q.Eq("Status", qr.Status))
	}

	sq := node.
		Select(
			q.And(where...),
			q.And(qr.fetch...),
		).
		OrderBy("Started").
		Reverse()

	err := sq.Find(&qr.Sessions)
	if err == storm.ErrNotFound {
		qr.Sessions = []History{}
		return nil
	}
	return err
}

type matcher func(History) bool

func (m matcher) Match(i interface{}) (bool, error) {
	return m(i.(History)), nil
}
