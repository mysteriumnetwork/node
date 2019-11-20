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
	"sort"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/identity"
)

// maxEntriesKeepPeriod describes how long entries are kept in memory storage.
// Older than this duration entries are removed on insert.
const maxEntriesKeepDuration = time.Hour * 24 * 30

// StatusStorage is responsible for status storage operations.
type StatusStorage interface {
	GetAllStatusEntries() []StatusEntry
	AddStatusEntry(msg StatusEntry)
}

// StatusEntry describes status entry.
type StatusEntry struct {
	PeerID       identity.Identity
	SessionID    string
	StatusCode   StatusCode
	Message      string
	CreatedAtUTC time.Time
}

// NewStatusStorage returns new StatusStorage instance.
func NewStatusStorage() StatusStorage {
	return &statusStorage{}
}

type statusStorage struct {
	entries    []StatusEntry
	entriesMux sync.RWMutex
}

func (s *statusStorage) GetAllStatusEntries() []StatusEntry {
	s.entriesMux.RLock()
	defer s.entriesMux.RUnlock()

	res := make([]StatusEntry, len(s.entries))
	copy(res, s.entries)

	// Sort by CreatedAtUTC descending to show newest entries first.
	sort.Slice(res, func(i, j int) bool {
		return res[i].CreatedAtUTC.After(res[j].CreatedAtUTC)
	})
	return res
}

func (s *statusStorage) AddStatusEntry(msg StatusEntry) {
	s.entriesMux.Lock()
	defer s.entriesMux.Unlock()

	// Remove old entries which are older that maxEntriesKeepDuration.
	var res []StatusEntry
	minValidEntryTime := time.Now().UTC().Add(-maxEntriesKeepDuration)
	for _, entry := range s.entries {
		if entry.CreatedAtUTC.After(minValidEntryTime) {
			res = append(res, entry)
		}
	}
	s.entries = res

	// Add new entry.
	s.entries = append(s.entries, msg)
}
