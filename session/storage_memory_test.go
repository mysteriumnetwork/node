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
	"fmt"
	"testing"
	"time"

	sessionEvent "github.com/mysteriumnetwork/node/session/event"
	"github.com/stretchr/testify/assert"
)

var (
	sessionExisting = Session{
		ID:   ID("mocked-id"),
		Last: true,
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
		map[ID]Session{sessionExisting.ID: sessionExisting, sessionNew.ID: sessionNew},
		storage.sessions,
	)
}

func TestStorage_GetAll(t *testing.T) {
	sessionFirst := Session{
		ID: ID("id1"),
	}
	sessionSecond := Session{
		ID:        ID("id2"),
		CreatedAt: time.Now(),
	}

	storage := &StorageMemory{
		sessions: map[ID]Session{
			sessionFirst.ID:  sessionFirst,
			sessionSecond.ID: sessionSecond,
		},
	}

	sessions := storage.GetAll()
	assert.Contains(t, sessions, sessionFirst)
	assert.Contains(t, sessions, sessionSecond)
}

func TestStorage_Remove(t *testing.T) {
	storage := mockStorage(sessionExisting)

	storage.Remove(sessionExisting.ID)
	assert.Len(t, storage.sessions, 0)
}

func TestStorage_RemoveNonExisting(t *testing.T) {
	storage := &StorageMemory{
		sessions:  map[ID]Session{},
		publisher: &mockPublisher{},
	}
	storage.Remove(sessionExisting.ID)
	assert.Len(t, storage.sessions, 0)
}

func TestStorage_Remove_Does_Not_Panic(t *testing.T) {
	id4 := ID("id4")
	storage := mockStorage(sessionExisting)
	sessionFirst := Session{ID: id4}
	sessionSecond := Session{ID: ID("id3")}
	storage.Add(sessionFirst)
	storage.Add(sessionSecond)
	storage.Remove(id4)
	storage.Remove(ID("id3"))
	assert.Len(t, storage.sessions, 1)
}

func TestStorage_PublishesEventsOnCreate(t *testing.T) {
	instance := expectedSession
	mp := &mockPublisher{}
	sessionStore := NewStorageMemory(mp)

	sessionStore.Add(instance)

	// since we're shooting the event in an asynchronous fashion, try every millisecond to see if we already have it
	attempts := 0
	for range time.After(time.Microsecond) {
		if attempts > 1000 {
			assert.Fail(t, "no change after a 1000 attempts")
			break
		}
		attempts++
		if mp.getLast().Action == sessionEvent.Created {
			break
		}
	}

	assert.Equal(t, sessionEvent.Payload{
		Action: sessionEvent.Created,
		ID:     string(expectedID),
	}, mp.published)
}

func TestStorage_PublishesEventsOnDelete(t *testing.T) {
	instance := expectedSession
	mp := &mockPublisher{}
	sessionStore := NewStorageMemory(mp)
	sessionStore.Add(instance)

	time.Sleep(time.Millisecond * 5)

	sessionStore.Remove(instance.ID)

	// since we're shooting the event in an asynchronous fashion, try every microsecond to see if we already have it
	attempts := 0

	for range time.After(time.Microsecond) {
		if attempts > 1000 {
			assert.Fail(t, "no change after a 1000 attempts")
			break
		}
		attempts++
		if mp.getLast().Action == sessionEvent.Removed {
			break
		}
	}

	assert.Equal(t, sessionEvent.Payload{
		Action: sessionEvent.Removed,
		ID:     string(expectedID),
	}, mp.published)
}

func mockStorage(sessionInstance Session) *StorageMemory {
	return &StorageMemory{
		sessions:  map[ID]Session{sessionInstance.ID: sessionInstance},
		publisher: &mockPublisher{},
	}
}

// to avoid compiler optimizing away our bench
var benchmarkStorageGetAllResult int

func Benchmark_Storage_GetAll(b *testing.B) {
	// Findings are as follows - with 100k sessions, we should be fine with a performance of 0.04s on my mac
	storage := NewStorageMemory(&mockPublisher{})
	sessionsToStore := 100000
	for i := 0; i < sessionsToStore; i++ {
		storage.Add(Session{ID: ID(fmt.Sprintf("ID%v", i)), CreatedAt: time.Now()})
	}

	var r int
	for n := 0; n < b.N; n++ {
		storedValues := storage.GetAll()
		r += len(storedValues)
	}
	benchmarkStorageGetAllResult = r
}
