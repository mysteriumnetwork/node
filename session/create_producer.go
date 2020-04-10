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
	"fmt"

	"github.com/mysteriumnetwork/node/communication"
)

type createProducer struct {
	Request CreateRequest
}

func (producer *createProducer) GetRequestEndpoint() communication.RequestEndpoint {
	return endpointSessionCreate
}

func (producer *createProducer) NewResponse() (responsePtr interface{}) {
	return &CreateResponse{}
}

func (producer *createProducer) Produce() (requestPtr interface{}) {
	return &producer.Request
}

// RequestSessionCreate requests session creation and returns session DTO
func RequestSessionCreate(sender communication.Sender, request CreateRequest) (CreateResponse, error) {
	responsePtr, err := sender.Request(&createProducer{Request: request})
	if err != nil {
		return CreateResponse{}, err
	}

	response := responsePtr.(*CreateResponse)
	if !response.Success {
		return CreateResponse{}, fmt.Errorf("session create failed: %s", response.Message)
	}

	return *response, nil
}

// AcknowledgeSession lets the provider know we've successfully established a connection
func AcknowledgeSession(sender communication.Sender, sessionID string) error {
	ack := NewAcknowledgeSender(sender)
	return ack.Send(AcknowledgeMessage{
		SessionID: sessionID,
	})
}
