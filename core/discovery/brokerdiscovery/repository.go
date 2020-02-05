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

	stopOnce sync.Once
	stopChan chan struct{}

	timeoutCheckStep  time.Duration
	watchdogLock      sync.Mutex
	timeoutCheckSeens map[market.ProposalID]time.Time
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

		stopChan:          make(chan struct{}),
		timeoutCheckStep:  proposalCheckInterval,
		timeoutCheckSeens: make(map[market.ProposalID]time.Time),
	}
}

// Proposal returns a single proposal by its ID.
func (r *Repository) Proposal(id market.ProposalID) (*market.ServiceProposal, error) {
	return r.storage.GetProposal(id)
}

// Proposals returns proposals matching the filter.
func (r *Repository) Proposals(filter *proposal.Filter) ([]market.ServiceProposal, error) {
	return r.storage.FindProposals(*filter)
}

// Start begins proposals synchronization to storage
func (r *Repository) Start() error {
	err := r.receiver.Receive(&registerConsumer{Callback: r.proposalRegisterMessage})
	if err != nil {
		return err
	}

	err = r.receiver.Receive(&unregisterConsumer{Callback: r.proposalUnregisterMessage})
	if err != nil {
		return err
	}

	err = r.receiver.Receive(&pingConsumer{Callback: r.proposalPingMessage})
	if err != nil {
		return err
	}

	go r.timeoutCheckLoop()

	return nil
}

// Stop ends proposals synchronization to storage
func (r *Repository) Stop() {
	r.stopOnce.Do(func() {
		close(r.stopChan)

		r.receiver.ReceiveUnsubscribe(pingEndpoint)
		r.receiver.ReceiveUnsubscribe(unregisterEndpoint)
		r.receiver.ReceiveUnsubscribe(registerEndpoint)
	})
}

func (r *Repository) proposalRegisterMessage(message registerMessage) error {
	r.storage.AddProposal(message.Proposal)

	r.watchdogLock.Lock()
	defer r.watchdogLock.Unlock()
	r.timeoutCheckSeens[message.Proposal.UniqueID()] = time.Now().UTC()

	return nil
}

func (r *Repository) proposalUnregisterMessage(message unregisterMessage) error {
	r.storage.RemoveProposal(message.Proposal.UniqueID())

	r.watchdogLock.Lock()
	defer r.watchdogLock.Unlock()
	delete(r.timeoutCheckSeens, message.Proposal.UniqueID())

	return nil
}

func (r *Repository) proposalPingMessage(message pingMessage) error {
	r.storage.AddProposal(message.Proposal)

	r.watchdogLock.Lock()
	defer r.watchdogLock.Unlock()
	r.timeoutCheckSeens[message.Proposal.UniqueID()] = time.Now()

	return nil
}

func (r *Repository) timeoutCheckLoop() {
	for {
		select {
		case <-r.stopChan:
			return
		case <-time.After(r.timeoutCheckStep):
			r.watchdogLock.Lock()
			for proposalID, proposalSeen := range r.timeoutCheckSeens {
				if time.Now().After(proposalSeen.Add(r.timeoutInterval)) {
					r.storage.RemoveProposal(proposalID)
					delete(r.timeoutCheckSeens, proposalID)
				}
			}
			r.watchdogLock.Unlock()
		}
	}
}
