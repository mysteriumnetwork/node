/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package connectivity

import (
	"sync"

	"github.com/mysteriumnetwork/node/identity"
)

//StatusStorage is status storage interface.
type StatusStorage interface {
	GetAllStatusEntries() []*StatusEntry
	AddStatusEntry(msg *StatusEntry)
}

// StatusEntry describes status entry.
type StatusEntry struct {
	PeerID     identity.Identity
	SessionID  string
	StatusCode StatusCode
	Message    string
}

// NewStatusStorage returns new StatusStorage instance.
func NewStatusStorage() StatusStorage {
	return &statusStorage{}
}

type statusStorage struct {
	entries    []*StatusEntry
	entriesMux sync.RWMutex
}

func (s *statusStorage) GetAllStatusEntries() []*StatusEntry {
	s.entriesMux.RLock()
	defer s.entriesMux.RUnlock()
	return s.entries
}

func (s *statusStorage) AddStatusEntry(msg *StatusEntry) {
	s.entriesMux.Lock()
	defer s.entriesMux.Unlock()

	// TODO: Maintain only last x entries to possibly to big memory usage.
	s.entries = append(s.entries, msg)
}
