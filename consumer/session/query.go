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
	"github.com/mysteriumnetwork/node/identity"
)

// NewStats initiates zero Stats instance.
func NewStats() Stats {
	return Stats{
		ConsumerCounts: make(map[identity.Identity]int),
	}
}

// Stats holds structure of aggregate session statistics.
type Stats struct {
	Count           int
	ConsumerCounts  map[identity.Identity]int
	SumDataSent     uint64
	SumDataReceived uint64
	SumDuration     time.Duration
	SumTokens       uint64
}

func (s *Stats) add(session History) {
	s.Count++

	if _, found := s.ConsumerCounts[session.ConsumerID]; !found {
		s.ConsumerCounts[session.ConsumerID] = 1
	} else {
		s.ConsumerCounts[session.ConsumerID]++
	}

	s.SumDataReceived += session.DataReceived
	s.SumDataSent += session.DataSent
	s.SumDuration += session.GetDuration()
	s.SumTokens += session.Tokens
}

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

	filterFrom      *time.Time
	filterTo        *time.Time
	filterDirection *string

	fetch []q.Matcher
}

// FilterFrom filters fetched sessions from given time.
func (qr *Query) FilterFrom(from time.Time) *Query {
	from = from.UTC()
	qr.filterFrom = &from
	return qr
}

// FilterTo filters fetched sessions to given time.
func (qr *Query) FilterTo(to time.Time) *Query {
	to = to.UTC()
	qr.filterTo = &to
	return qr
}

// FilterDirection filters fetched sessions by direction.
func (qr *Query) FilterDirection(direction string) *Query {
	qr.filterDirection = &direction
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
			qr.Stats.add(session)
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
	if qr.filterFrom != nil && qr.filterTo != nil {
		for i := qr.filterFrom.Truncate(stepDay); !i.After(*qr.filterTo); i = i.Add(stepDay) {
			qr.StatsByDay[i] = NewStats()
		}
	}

	qr.fetch = append(
		qr.fetch,
		matcher(func(session History) bool {
			i := session.Started.Truncate(stepDay)

			stats := qr.StatsByDay[i]
			stats.add(session)
			qr.StatsByDay[i] = stats
			return true
		}),
	)

	return qr
}

func (qr *Query) run(node storm.Node) error {
	where := make([]q.Matcher, 0)
	if qr.filterFrom != nil {
		where = append(where, q.Gte("Started", qr.filterFrom))
	}
	if qr.filterTo != nil {
		where = append(where, q.Lte("Started", qr.filterTo))
	}
	if qr.filterDirection != nil {
		where = append(where, q.Eq("Direction", qr.filterDirection))
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
