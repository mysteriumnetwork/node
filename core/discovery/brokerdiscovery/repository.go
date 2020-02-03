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

package brokerdiscovery

import (
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/market"
)

// Repository provides proposals from the broker.
type Repository struct {
	storage         *ProposalStorage
	receiver        communication.Receiver
	timeoutInterval time.Duration

	watchdogStop chan struct{}
	watchdogStep time.Duration
	watchdogLock sync.Mutex
	watchdogSeen map[market.ProposalID]time.Time
}

// NewRepository constructs a new proposal repository (backed by the broker).
func NewRepository(
	connection nats.Connection,
	eventPublisher eventbus.Publisher,
	proposalTimeoutInterval time.Duration,
	proposalCheckInterval time.Duration,
) *Repository {
	return &Repository{
		storage:         NewStorage(eventPublisher),
		receiver:        nats.NewReceiver(connection, communication.NewCodecJSON(), "*"),
		timeoutInterval: proposalTimeoutInterval,

		watchdogStop: make(chan struct{}),
		watchdogStep: proposalCheckInterval,
		watchdogSeen: make(map[market.ProposalID]time.Time),
	}
}

// Proposal returns a single proposal by its ID.
func (s *Repository) Proposal(id market.ProposalID) (*market.ServiceProposal, error) {
	return s.storage.GetProposal(id)
}

// Proposals returns proposals matching the filter.
func (s *Repository) Proposals(filter *proposal.Filter) ([]market.ServiceProposal, error) {
	return s.storage.FindProposals(*filter)
}

// Start begins proposals synchronization to storage
func (s *Repository) Start() error {
	err := s.receiver.Receive(&registerConsumer{Callback: s.proposalRegisterMessage})
	if err != nil {
		return err
	}

	err = s.receiver.Receive(&unregisterConsumer{Callback: s.proposalUnregisterMessage})
	if err != nil {
		return err
	}

	err = s.receiver.Receive(&pingConsumer{Callback: s.proposalPingMessage})
	if err != nil {
		return err
	}

	go s.proposalWatchdog()

	return nil
}

// Stop ends proposals synchronization to storage
func (s *Repository) Stop() {
	s.watchdogStop <- struct{}{}

	s.receiver.ReceiveUnsubscribe(pingEndpoint)
	s.receiver.ReceiveUnsubscribe(unregisterEndpoint)
	s.receiver.ReceiveUnsubscribe(registerEndpoint)
}

func (s *Repository) proposalRegisterMessage(message registerMessage) error {
	s.storage.AddProposal(message.Proposal)

	s.watchdogLock.Lock()
	defer s.watchdogLock.Unlock()
	s.watchdogSeen[message.Proposal.UniqueID()] = time.Now().UTC()

	return nil
}

func (s *Repository) proposalUnregisterMessage(message unregisterMessage) error {
	s.storage.RemoveProposal(message.Proposal.UniqueID())

	s.watchdogLock.Lock()
	defer s.watchdogLock.Unlock()
	delete(s.watchdogSeen, message.Proposal.UniqueID())

	return nil
}

func (s *Repository) proposalPingMessage(message pingMessage) error {
	s.storage.AddProposal(message.Proposal)

	s.watchdogLock.Lock()
	defer s.watchdogLock.Unlock()
	s.watchdogSeen[message.Proposal.UniqueID()] = time.Now()

	return nil
}

func (s *Repository) proposalWatchdog() {
	for {
		select {
		case <-s.watchdogStop:
			return
		case <-time.After(s.watchdogStep):
			s.watchdogLock.Lock()
			for proposalID, proposalSeen := range s.watchdogSeen {
				if time.Now().After(proposalSeen.Add(s.timeoutInterval)) {
					s.storage.RemoveProposal(proposalID)
					delete(s.watchdogSeen, proposalID)
				}
			}
			s.watchdogLock.Unlock()
		}
	}
}
