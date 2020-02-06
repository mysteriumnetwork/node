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

package session

import (
	"testing"
	"time"

	sessionEvent "github.com/mysteriumnetwork/node/session/event"
	"github.com/stretchr/testify/assert"
)

func TestEventBasedStorage_PublishesEventsOnCreate(t *testing.T) {
	instance := expectedSession
	mp := &mockPublisher{}
	sessionStore := NewEventBasedStorage(mp, NewStorageMemory())

	sessionStore.Add(instance)

	// since we're shooting the event in an asynchronous fashion, try every millisecond to see if we already have it
	attempts := 0
	for range time.After(time.Millisecond) {
		if attempts > 50 {
			assert.Fail(t, "no change after a 50 attempts")
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

func TestEventBasedStorage_PublishesEventsOnDelete(t *testing.T) {
	instance := expectedSession
	mp := &mockPublisher{}
	sessionStore := NewEventBasedStorage(mp, NewStorageMemory())
	sessionStore.Add(instance)

	time.Sleep(time.Millisecond * 5)

	sessionStore.Remove(instance.ID)

	// since we're shooting the event in an asynchronous fashion, try every microsecond to see if we already have it
	attempts := 0

	for range time.After(time.Millisecond) {
		if attempts > 50 {
			assert.Fail(t, "no change after a 50 attempts")
			break
		}
		attempts++
		t.Log("attempts", attempts)
		if mp.getLast().Action == sessionEvent.Removed {
			break
		}
	}

	assert.Equal(t, sessionEvent.Payload{
		Action: sessionEvent.Removed,
		ID:     string(expectedID),
	}, mp.published)
}

func TestEventBasedStorage_PublishesEventsOnDataTransferUpdate(t *testing.T) {
	instance := expectedSession
	mp := &mockPublisher{}
	sessionStore := NewEventBasedStorage(mp, NewStorageMemory())
	sessionStore.Add(instance)

	time.Sleep(time.Millisecond * 5)

	sessionStore.UpdateDataTransfer(instance.ID, 1, 2)

	// since we're shooting the event in an asynchronous fashion, try every microsecond to see if we already have it
	attempts := 0

	for range time.After(time.Millisecond) {
		if attempts > 50 {
			assert.Fail(t, "no change after a 50 attempts")
			break
		}
		attempts++
		t.Log("attempts", attempts)
		if mp.getLast().Action == sessionEvent.Updated {
			break
		}
	}

	assert.Equal(t, sessionEvent.Payload{
		Action: sessionEvent.Updated,
		ID:     string(expectedID),
	}, mp.published)
}

func TestNewEventBasedStorage_HandlesAppEventTokensEarned(t *testing.T) {
	// given
	session := expectedSession
	mp := &mockPublisher{}
	sessionStore := NewEventBasedStorage(mp, NewStorageMemory())
	sessionStore.Add(session)

	storedSession, ok := sessionStore.Find(session.ID)
	assert.True(t, ok)
	assert.Zero(t, storedSession.TokensEarned)

	// when
	sessionStore.consumeTokensEarnedEvent(sessionEvent.AppEventSessionTokensEarned{
		Consumer: consumerID,
		Total:    500,
	})
	// then
	storedSession, ok = sessionStore.Find(session.ID)
	assert.True(t, ok)
	assert.EqualValues(t, 500, storedSession.TokensEarned)
	assert.Eventually(t, func() bool {
		evt := mp.getLast()
		return evt.ID == string(session.ID) && evt.Action == sessionEvent.Updated
	}, 1*time.Second, 5*time.Millisecond)
}

func TestEventBasedStorage_PublishesEventsOnRemoveForService(t *testing.T) {
	instance := expectedSession
	mp := &mockPublisher{}
	sessionStore := NewEventBasedStorage(mp, NewStorageMemory())
	sessionStore.Add(instance)

	time.Sleep(time.Millisecond * 5)

	sessionStore.RemoveForService("whatever")

	// since we're shooting the event in an asynchronous fashion, try every microsecond to see if we already have it
	attempts := 0

	for range time.After(time.Millisecond) {
		if attempts > 50 {
			assert.Fail(t, "no change after a 50 attempts")
			break
		}
		attempts++
		t.Log("attempts", attempts)
		if mp.getLast().Action == sessionEvent.Removed {
			break
		}
	}

	assert.Equal(t, sessionEvent.Payload{
		Action: sessionEvent.Removed,
		ID:     "",
	}, mp.published)
}
