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
	"github.com/mysteriumnetwork/node/session/event"
)

type eventPublisher interface {
	Publish(topic string, data interface{})
	SubscribeAsync(topic string, f interface{}) error
}

type storage interface {
	Add(sessionInstance Session)
	GetAll() []Session
	UpdateDataTransfer(id ID, up, down uint64)
	UpdateEarnings(id ID, total uint64)
	Find(id ID) (Session, bool)
	FindBy(opts FindOpts) (ID, bool)
	Remove(id ID)
	RemoveForService(serviceID string)
}

// EventBasedStorage wraps storage methods and adds relevant event pub/sub where required
type EventBasedStorage struct {
	bus     eventPublisher
	storage storage
}

// NewEventBasedStorage returns a new instance of event based storage
func NewEventBasedStorage(bus eventPublisher, storage storage) *EventBasedStorage {
	return &EventBasedStorage{
		bus:     bus,
		storage: storage,
	}
}

// Add adds a session and publishes a creation event
func (ebs *EventBasedStorage) Add(sessionInstance Session) {
	ebs.storage.Add(sessionInstance)
	go ebs.bus.Publish(event.AppTopicSession, event.Payload{
		ID:     string(sessionInstance.ID),
		Action: event.Created,
	})
}

// GetAll returns all sessions
func (ebs *EventBasedStorage) GetAll() []Session {
	return ebs.storage.GetAll()
}

func (ebs *EventBasedStorage) consumeDataTransferredEvent(e event.AppEventDataTransferred) {
	// From a server perspective, bytes up are the actual bytes the client downloaded(aka the bytes we pushed to the consumer)
	// To lessen the confusion, I suggest having the bytes reversed on the session instance.
	// This way, the session will show that it downloaded the bytes in a manner that is easier to comprehend.
	ebs.UpdateDataTransfer(ID(e.ID), e.Down, e.Up)
}

func (ebs *EventBasedStorage) consumeTokensEarnedEvent(e event.AppEventSessionTokensEarned) {
	ebs.storage.UpdateEarnings(ID(e.SessionID), e.Total)
	go ebs.bus.Publish(event.AppTopicSession, event.Payload{
		ID:     e.SessionID,
		Action: event.Updated,
	})
}

// UpdateDataTransfer updates the data transfer for a session
func (ebs *EventBasedStorage) UpdateDataTransfer(id ID, up, down uint64) {
	ebs.storage.UpdateDataTransfer(id, up, down)
	go ebs.bus.Publish(event.AppTopicSession, event.Payload{
		ID:     string(id),
		Action: event.Updated,
	})
}

// Find finds a session
func (ebs *EventBasedStorage) Find(id ID) (Session, bool) {
	return ebs.storage.Find(id)
}

// FindBy returns a session by find options.
func (ebs *EventBasedStorage) FindBy(opts FindOpts) (ID, bool) {
	return ebs.storage.FindBy(opts)
}

// Remove removes the session and publishes a removal event
func (ebs *EventBasedStorage) Remove(id ID) {
	ebs.storage.Remove(id)
	go ebs.bus.Publish(event.AppTopicSession, event.Payload{
		ID:     string(id),
		Action: event.Removed,
	})
}

// RemoveForService removes all the sessions for a service and publishes a delete event
func (ebs *EventBasedStorage) RemoveForService(serviceID string) {
	ebs.storage.RemoveForService(serviceID)
	go ebs.bus.Publish(event.AppTopicSession, event.Payload{
		ID:     "",
		Action: event.Removed,
	})
}

// Subscribe subscribes the ebs to relevant events
func (ebs *EventBasedStorage) Subscribe() error {
	if err := ebs.bus.SubscribeAsync(event.AppTopicDataTransferred, ebs.consumeDataTransferredEvent); err != nil {
		return err
	}
	if err := ebs.bus.SubscribeAsync(event.AppTopicSessionTokensEarned, ebs.consumeTokensEarnedEvent); err != nil {
		return err
	}
	return nil
}
