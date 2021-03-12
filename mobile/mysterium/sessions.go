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

	h, err := mb.sessionStorage.List(f)
	if err != nil {
		return nil, err
	}
	return json.Marshal(h)
}
