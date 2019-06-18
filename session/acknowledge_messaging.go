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
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/identity"
)

// AcknowledgeRequest represents the acknowledge request
type AcknowledgeRequest struct {
	AcknowledgeMessage AcknowledgeMessage `json:"acknowledgeMessage"`
}

// AcknowledgeMessage represents the acknowledge payload
type AcknowledgeMessage struct {
	SessionID string `json:"sessionId"`
}

// Acknowledger performs the actual acknowledging
type Acknowledger interface {
	Acknowledge(cID identity.Identity, sessionID string) error
}

const endpointAcknowledge = "session-aknowledge"
const acknowledgeEndpoint = communication.MessageEndpoint(endpointAcknowledge)

// AcknowledgeSender sends session acknowledge messages
type AcknowledgeSender struct {
	sender communication.Sender
}

// NewAcknowledgeSender creates a new instance of acknowledge sender
func NewAcknowledgeSender(sender communication.Sender) *AcknowledgeSender {
	return &AcknowledgeSender{
		sender: sender,
	}
}

// Send sends the acknowledge message
func (as *AcknowledgeSender) Send(am AcknowledgeMessage) error {
	return as.sender.Send(&acknowledgeMessageProducer{AcknowledgeMessage: am})
}

// AcknowledgeListener allows us to listen for acknowledge messages from consumers
type AcknowledgeListener struct {
	acknowledgeMessageConsumer *acknowledgeMessageConsumer
}

// NewListener returns a new instance of AcknowledgeListener
func NewListener() *AcknowledgeListener {
	return &AcknowledgeListener{
		acknowledgeMessageConsumer: &acknowledgeMessageConsumer{},
	}
}

// GetConsumer boilerplate for communication
func (al *AcknowledgeListener) GetConsumer() *acknowledgeMessageConsumer {
	return al.acknowledgeMessageConsumer
}

// Consume handles requests from endpoint and replies with response
func (amc *acknowledgeMessageConsumer) Consume(requestPtr interface{}) (err error) {
	request := requestPtr.(*AcknowledgeRequest)

	err = amc.Acknowledger.Acknowledge(amc.PeerID, request.AcknowledgeMessage.SessionID)
	return err
}

// Dialog boilerplate below, please ignore

type acknowledgeMessageConsumer struct {
	Acknowledger Acknowledger
	PeerID       identity.Identity
}

// GetMessageEndpoint returns endpoint where to receive messages
func (amc *acknowledgeMessageConsumer) GetMessageEndpoint() communication.MessageEndpoint {
	return acknowledgeEndpoint
}

// NewRequest creates struct where request from endpoint will be serialized
func (amc *acknowledgeMessageConsumer) NewMessage() (requestPtr interface{}) {
	return &AcknowledgeRequest{}
}

// balanceMessageProducer
type acknowledgeMessageProducer struct {
	AcknowledgeMessage AcknowledgeMessage
}

// GetMessageEndpoint returns endpoint where to receive messages
func (amp *acknowledgeMessageProducer) GetMessageEndpoint() communication.MessageEndpoint {
	return acknowledgeEndpoint
}

// Produce produces a request message
func (amp *acknowledgeMessageProducer) Produce() (requestPtr interface{}) {
	return &AcknowledgeRequest{
		AcknowledgeMessage: amp.AcknowledgeMessage,
	}
}

// NewResponse returns a new response object
func (amp *acknowledgeMessageProducer) NewResponse() (responsePtr interface{}) {
	return nil
}
