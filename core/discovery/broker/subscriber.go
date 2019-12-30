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
	"time"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/core/discovery"
	"github.com/mysteriumnetwork/node/market"
)

// proposalSubscriber responsible for handling proposal events through Broker (Mysterium Communication)
type proposalSubscriber struct {
	proposalStorage *discovery.ProposalStorage
	receiver        communication.Receiver
}

// NewProposalSubscriber returns new proposalSubscriber instance.
func NewProposalSubscriber(proposalsStorage *discovery.ProposalStorage, connection nats.Connection) *proposalSubscriber {
	return &proposalSubscriber{
		proposalStorage: proposalsStorage,
		receiver:        nats.NewReceiver(connection, communication.NewCodecJSON(), "*"),
	}
}

// Start begins proposals synchronisation to storage
func (s *proposalSubscriber) Start() error {
	errSubscribe := s.receiver.Receive(&registerConsumer{Callback: s.proposalRegisterMessage})
	if errSubscribe != nil {
		return errSubscribe
	}

	errSubscribe = s.receiver.Receive(&unregisterConsumer{Callback: s.proposalUnregisterMessage})
	if errSubscribe != nil {
		return errSubscribe
	}

	return nil
}

// Stop ends proposals synchronisation to storage
func (s *proposalSubscriber) Stop() {}

func (s *proposalSubscriber) proposalRegisterMessage(message registerMessage) error {
	s.proposalStorage.AddProposal(message.Proposal)
	go s.idleProposalUnregister(message.Proposal)

	return nil
}

func (s *proposalSubscriber) proposalUnregisterMessage(message unregisterMessage) error {
	s.proposalStorage.RemoveProposal(message.Proposal)
	return nil
}

func (s *proposalSubscriber) idleProposalUnregister(proposal market.ServiceProposal) {
	<-time.After(2 * time.Minute)
	s.proposalStorage.RemoveProposal(proposal)
}
