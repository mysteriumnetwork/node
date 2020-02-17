/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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
	"encoding/json"
	"fmt"

	"github.com/mysteriumnetwork/node/communication"
)

type createProducer struct {
	ProposalID   int
	Config       json.RawMessage
	ConsumerInfo *ConsumerInfo
}

func (producer *createProducer) GetRequestEndpoint() communication.RequestEndpoint {
	return endpointSessionCreate
}

func (producer *createProducer) NewResponse() (responsePtr interface{}) {
	return &CreateResponse{}
}

func (producer *createProducer) Produce() (requestPtr interface{}) {
	return &CreateRequest{
		ProposalID:   producer.ProposalID,
		Config:       producer.Config,
		ConsumerInfo: producer.ConsumerInfo,
	}
}

// RequestSessionCreate requests session creation and returns session DTO
func RequestSessionCreate(sender communication.Sender, proposalID int, config interface{}, ci ConsumerInfo) (session SessionDto, pi PaymentInfo, err error) {
	sessionCreateConfigJSON, err := json.Marshal(config)
	if err != nil {
		return
	}

	responsePtr, err := sender.Request(&createProducer{
		ProposalID:   proposalID,
		Config:       sessionCreateConfigJSON,
		ConsumerInfo: &ci,
	})
	if err != nil {
		return
	}

	response := responsePtr.(*CreateResponse)
	if !response.Success {
		err = fmt.Errorf("session create failed: %s", response.Message)
		return
	}

	session = SessionDto{
		ID:     response.Session.ID,
		Config: response.Session.Config,
	}
	pi = response.PaymentInfo
	return
}

// AcknowledgeSession lets the provider know we've successfully established a connection
func AcknowledgeSession(sender communication.Sender, sessionID string) error {
	ack := NewAcknowledgeSender(sender)
	return ack.Send(AcknowledgeMessage{
		SessionID: sessionID,
	})
}
