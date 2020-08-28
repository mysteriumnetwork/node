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

package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/session"
	sessionEvent "github.com/mysteriumnetwork/node/session/event"
	"github.com/mysteriumnetwork/node/trace"
	"github.com/stretchr/testify/assert"
)

var (
	sessionExisting, _ = NewSession(
		&Instance{ID: "1"},
		&pb.SessionRequest{Consumer: &pb.ConsumerInfo{Id: "deadbeef"}},
		trace.NewTracer(""),
	)
)

func TestSessionPool_FindSession_Existing(t *testing.T) {
	pool := mockPool(mocks.NewEventBus(), sessionExisting)

	sessionInstance, found := pool.Find(sessionExisting.ID)

	assert.True(t, found)
	assert.Exactly(t, sessionExisting, sessionInstance)
}

func TestSessionPool_FindSession_Unknown(t *testing.T) {
	storage := mockPool(mocks.NewEventBus(), sessionExisting)

	sessionInstance, found := storage.Find(session.ID("unknown-id"))
	assert.False(t, found)
	assert.Nil(t, sessionInstance)
}

func TestSessionPool_Add(t *testing.T) {
	pool := mockPool(mocks.NewEventBus(), sessionExisting)
	sessionNew, _ := NewSession(&Instance{}, &pb.SessionRequest{}, trace.NewTracer(""))

	pool.Add(sessionNew)
	assert.Exactly(
		t,
		map[session.ID]*Session{sessionExisting.ID: sessionExisting, sessionNew.ID: sessionNew},
		pool.sessions,
	)
}

func TestSessionPool_Add_PublishesEvents(t *testing.T) {
	// given
	session, _ := NewSession(&Instance{}, &pb.SessionRequest{}, trace.NewTracer(""))
	mp := mocks.NewEventBus()
	pool := NewSessionPool(mp)

	// when
	pool.Add(session)

	// then
	assert.Eventually(t, lastEventMatches(mp, session.ID, sessionEvent.CreatedStatus), 2*time.Second, 10*time.Millisecond)
}

func TestSessionPool_FindByPeer(t *testing.T) {
	pool := mockPool(mocks.NewEventBus(), sessionExisting)
	session, ok := pool.FindBy(FindOpts{&sessionExisting.ConsumerID, ""})
	assert.True(t, ok)
	assert.Equal(t, sessionExisting.ID, session.ID)
}

func TestSessionPool_GetAll(t *testing.T) {
	sessionFirst, _ := NewSession(&Instance{}, &pb.SessionRequest{}, trace.NewTracer(""))
	sessionSecond, _ := NewSession(&Instance{}, &pb.SessionRequest{}, trace.NewTracer(""))

	pool := &SessionPool{
		sessions: map[session.ID]*Session{
			sessionFirst.ID:  sessionFirst,
			sessionSecond.ID: sessionSecond,
		},
	}

	sessions := pool.GetAll()
	assert.Contains(t, sessions, sessionFirst)
	assert.Contains(t, sessions, sessionSecond)
}

func TestSessionPool_Remove(t *testing.T) {
	pool := mockPool(mocks.NewEventBus(), sessionExisting)

	pool.Remove(sessionExisting.ID)
	assert.Len(t, pool.sessions, 0)
}

func TestSessionPool_RemoveNonExisting(t *testing.T) {
	pool := &SessionPool{
		sessions:  map[session.ID]*Session{},
		publisher: mocks.NewEventBus(),
	}
	pool.Remove(sessionExisting.ID)
	assert.Len(t, pool.sessions, 0)
}

func TestSessionPool_Remove_Does_Not_Panic(t *testing.T) {
	pool := mockPool(mocks.NewEventBus(), sessionExisting)

	sessionFirst, _ := NewSession(&Instance{}, &pb.SessionRequest{}, trace.NewTracer(""))
	pool.Add(sessionFirst)

	sessionSecond, _ := NewSession(&Instance{}, &pb.SessionRequest{}, trace.NewTracer(""))
	pool.Add(sessionSecond)

	pool.Remove(sessionFirst.ID)
	pool.Remove(sessionSecond.ID)
	assert.Len(t, pool.sessions, 1)
}

func TestSessionPool_Remove_PublishesEvents(t *testing.T) {
	// given
	mp := mocks.NewEventBus()
	pool := mockPool(mp, sessionExisting)

	// when
	pool.Remove(sessionExisting.ID)

	// then
	assert.Eventually(t, lastEventMatches(mp, sessionExisting.ID, sessionEvent.RemovedStatus), 2*time.Second, 10*time.Millisecond)
}

func TestSessionPool_RemoveForService_PublishesEvents(t *testing.T) {
	// given
	mp := mocks.NewEventBus()
	pool := mockPool(mp, sessionExisting)

	// when
	pool.RemoveForService(sessionExisting.ServiceID)

	// then
	assert.Eventually(t, lastEventMatches(mp, sessionExisting.ID, sessionEvent.RemovedStatus), 2*time.Second, 10*time.Millisecond)
}

func mockPool(publisher publisher, sessionInstance *Session) *SessionPool {
	return &SessionPool{
		sessions:  map[session.ID]*Session{sessionInstance.ID: sessionInstance},
		publisher: publisher,
	}
}

func lastEventMatches(mp *mocks.EventBus, id session.ID, action sessionEvent.Status) func() bool {
	return func() bool {
		last := mp.Pop()
		evt, ok := last.(sessionEvent.AppEventSession)
		if !ok {
			return false
		}
		return evt.Session.ID == string(id) && evt.Status == action
	}
}

// to avoid compiler optimizing away our bench
var benchmarkSessionPoolGetAllResult int

func Benchmark_SessionPool_GetAll(b *testing.B) {
	// Findings are as follows - with 100k sessions, we should be fine with a performance of 0.04s on my mac
	pool := NewSessionPool(mocks.NewEventBus())
	sessionsToStore := 100000
	for i := 0; i < sessionsToStore; i++ {
		pool.Add(&Session{ID: session.ID(fmt.Sprintf("ID%v", i)), CreatedAt: time.Now()})
	}

	var r int
	for n := 0; n < b.N; n++ {
		storedValues := pool.GetAll()
		r += len(storedValues)
	}
	benchmarkSessionPoolGetAllResult = r
}
