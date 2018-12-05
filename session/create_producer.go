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
	"errors"

	"github.com/mysteriumnetwork/node/communication"
)

type createProducer struct {
	ProposalID int
}

func (producer *createProducer) GetRequestEndpoint() communication.RequestEndpoint {
	return endpointSessionCreate
}

func (producer *createProducer) NewResponse() (responsePtr interface{}) {
	return &CreateResponse{}
}

func (producer *createProducer) Produce() (requestPtr interface{}) {
	return &CreateRequest{
		ProposalId: producer.ProposalID,
	}
}

// AckHandler allows the services to handle acks in their prefered way
type AckHandler func(sessionResponse SessionDto, ackSend func(payload interface{}) error) (json.RawMessage, error)

// RequestSessionCreate requests session creation and returns session DTO
func RequestSessionCreate(sender communication.Sender, proposalID int, ackHandler AckHandler) (sessionID ID, sessionConfig json.RawMessage, err error) {
	responsePtr, err := sender.Request(&createProducer{
		ProposalID: proposalID,
	})
	if err != nil {
		return
	}

	response := responsePtr.(*CreateResponse)
	if !response.Success {
		err = errors.New("Session create failed. " + response.Message)
		return
	}

	acker := func(payload interface{}) error {
		_, err := sender.Request(&AckProducer{
			Payload: payload,
		})
		return err
	}

	config, err := ackHandler(response.Session, acker)
	if err != nil {
		err = errors.New("Session ack failed. " + err.Error())
		return
	}

	sessionID = response.Session.ID
	sessionConfig = config
	return
}
