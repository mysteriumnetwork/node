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

package session

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	sessionExisting = Session{
		ID: ID("mocked-id"),
	}
)

func TestStorage_FindSession_Existing(t *testing.T) {
	storage := mockStorage(sessionExisting)

	sessionInstance, found := storage.Find(sessionExisting.ID)

	assert.True(t, found)
	assert.Exactly(t, sessionExisting, sessionInstance)
}

func TestStorage_FindSession_Unknown(t *testing.T) {
	storage := mockStorage(sessionExisting)

	sessionInstance, found := storage.Find(ID("unknown-id"))
	assert.False(t, found)
	assert.Exactly(t, Session{}, sessionInstance)
}

func TestStorage_Add(t *testing.T) {
	storage := mockStorage(sessionExisting)
	sessionNew := Session{
		ID: ID("new-id"),
	}

	storage.Add(sessionNew)
	assert.Exactly(
		t,
		[]Session{sessionExisting, sessionNew},
		storage.sessions,
	)
	assert.Exactly(
		t,
		map[ID]int{
			sessionExisting.ID: 0,
			sessionNew.ID:      1,
		},
		storage.idToIndex,
	)
}

func TestStorage_GetAll(t *testing.T) {
	sessionFirst := Session{ID: ID("id1")}
	sessionSecond := Session{ID: ID("id2")}
	storage := &StorageMemory{
		sessions: []Session{
			sessionFirst,
			sessionSecond,
		},
		idToIndex: map[ID]int{
			sessionFirst.ID:  0,
			sessionSecond.ID: 1,
		},
	}

	sessions, err := storage.GetAll()
	assert.NoError(t, err)
	assert.Equal(t, sessions, []Session{sessionFirst, sessionSecond})
}

func TestStorage_Remove(t *testing.T) {
	storage := mockStorage(sessionExisting)

	storage.Remove(sessionExisting.ID)
	assert.Len(t, storage.sessions, 0)
	assert.Len(t, storage.idToIndex, 0)
}

func mockStorage(sessionInstance Session) *StorageMemory {
	return &StorageMemory{
		sessions: []Session{sessionInstance},
		idToIndex: map[ID]int{
			sessionInstance.ID: 0,
		},
	}
}
