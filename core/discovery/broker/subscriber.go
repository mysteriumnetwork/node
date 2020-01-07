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

package broker

import (
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/core/discovery"
	"github.com/mysteriumnetwork/node/market"
)

// ProposalSubscriber responsible for handling proposal events through Broker (Mysterium Communication)
type ProposalSubscriber struct {
	proposalStorage  *discovery.ProposalStorage
	proposalIdleness time.Duration
	receiver         communication.Receiver

	watchdogStep time.Duration
	watchdogStop chan bool
	watchdogLock sync.Mutex
	watchdogSeen map[market.ProposalID]time.Time
}

// NewProposalSubscriber returns new ProposalSubscriber instance.
func NewProposalSubscriber(
	proposalsStorage *discovery.ProposalStorage,
	proposalIdleness time.Duration,
	connection nats.Connection,
) *ProposalSubscriber {
	return &ProposalSubscriber{
		proposalStorage:  proposalsStorage,
		proposalIdleness: proposalIdleness,
		receiver:         nats.NewReceiver(connection, communication.NewCodecJSON(), "*"),

		watchdogStep: proposalIdleness / 10,
		watchdogStop: make(chan bool),
		watchdogSeen: make(map[market.ProposalID]time.Time),
	}
}

// Start begins proposals synchronization to storage
func (s *ProposalSubscriber) Start() error {
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
func (s *ProposalSubscriber) Stop() {
	close(s.watchdogStop)
}

func (s *ProposalSubscriber) proposalRegisterMessage(message registerMessage) error {
	s.proposalStorage.AddProposal(message.Proposal)

	s.watchdogLock.Lock()
	defer s.watchdogLock.Unlock()
	s.watchdogSeen[message.Proposal.UniqueID()] = time.Now().UTC()

	return nil
}

func (s *ProposalSubscriber) proposalUnregisterMessage(message unregisterMessage) error {
	s.proposalStorage.RemoveProposal(message.Proposal.UniqueID())

	s.watchdogLock.Lock()
	defer s.watchdogLock.Unlock()
	delete(s.watchdogSeen, message.Proposal.UniqueID())

	return nil
}

func (s *ProposalSubscriber) proposalPingMessage(message pingMessage) error {
	s.proposalStorage.AddProposal(message.Proposal)

	s.watchdogLock.Lock()
	defer s.watchdogLock.Unlock()
	s.watchdogSeen[message.Proposal.UniqueID()] = time.Now()

	return nil
}

func (s *ProposalSubscriber) proposalWatchdog() {
	for {
		select {
		case <-s.watchdogStop:
			return
		case <-time.After(s.watchdogStep):
			s.watchdogLock.Lock()
			for proposalID, proposalSeen := range s.watchdogSeen {
				if time.Now().After(proposalSeen.Add(s.proposalIdleness)) {
					s.proposalStorage.RemoveProposal(proposalID)
					delete(s.watchdogSeen, proposalID)
				}
			}
			s.watchdogLock.Unlock()
		}
	}
}
