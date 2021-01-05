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
	"sync"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/event"
)

// NewSessionPool initiates new session storage
func NewSessionPool(publisher publisher) *SessionPool {
	sm := &SessionPool{
		sessions:  make(map[session.ID]*Session),
		lock:      sync.Mutex{},
		publisher: publisher,
	}
	return sm
}

// SessionPool maintains all current sessions in memory
type SessionPool struct {
	sessions  map[session.ID]*Session
	lock      sync.Mutex
	publisher publisher
}

// Add puts given session to storage and publishes a creation event.
// Multiple sessions per peerID is possible in case different services are used
func (sp *SessionPool) Add(instance *Session) {
	sp.lock.Lock()
	defer sp.lock.Unlock()

	sp.sessions[instance.ID] = instance
	sp.publisher.Publish(event.AppTopicSession, instance.toEvent(event.CreatedStatus))
}

// GetAll returns all sessions in storage
func (sp *SessionPool) GetAll() []*Session {
	sp.lock.Lock()
	defer sp.lock.Unlock()

	// we're never gonna have more than 100000 sessions ongoing on a single node - performance here should not be an issue.
	// see Benchmark_Storage_GetAll
	sessions := make([]*Session, len(sp.sessions))

	i := 0
	for _, value := range sp.sessions {
		sessions[i] = value
		i++
	}
	return sessions
}

// Find returns underlying session instance
func (sp *SessionPool) Find(id session.ID) (*Session, bool) {
	sp.lock.Lock()
	defer sp.lock.Unlock()

	instance, found := sp.sessions[id]
	return instance, found
}

// FindOpts provides fields to search sessions.
type FindOpts struct {
	Peer        *identity.Identity
	ServiceType string
}

// FindBy returns a session by find options.
func (sp *SessionPool) FindBy(opts FindOpts) (*Session, bool) {
	sp.lock.Lock()
	defer sp.lock.Unlock()

	for _, session := range sp.sessions {
		if opts.Peer != nil && *opts.Peer != session.ConsumerID {
			continue
		}
		if opts.ServiceType != "" && opts.ServiceType != session.Proposal.ServiceType {
			continue
		}
		return session, true
	}
	return nil, false
}

// Remove removes given session from underlying storage
func (sp *SessionPool) Remove(id session.ID) {
	sp.lock.Lock()
	defer sp.lock.Unlock()

	if instance, found := sp.sessions[id]; found {
		delete(sp.sessions, id)
		go sp.publisher.Publish(event.AppTopicSession, instance.toEvent(event.RemovedStatus))
	}
}

// RemoveForService removes all sessions which belong to given service
func (sp *SessionPool) RemoveForService(serviceID string) {
	sessions := sp.GetAll()
	for _, session := range sessions {
		if session.ServiceID == serviceID {
			sp.Remove(session.ID)
		}
	}
}
