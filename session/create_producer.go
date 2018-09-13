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

type CreateProducer struct {
	ProposalID int
}

func (producer *CreateProducer) GetRequestEndpoint() communication.RequestEndpoint {
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

// RequestSessionCreate requests session creation and returns session DTO
func RequestSessionCreate(sender communication.Sender, proposalID int, sessionPtr *Session) error {
	responsePtr, err := sender.Request(&createProducer{
		ProposalID: proposalID,
	})
	response := responsePtr.(*CreateResponse)

	if err != nil || !response.Success {
		return errors.New("Session create failed. " + response.Message)
	}

	return responseToSession(response, sessionPtr)
}

func responseToSession(response *CreateResponse, sessionPtr *Session) error {
	sessionPtr.ID = response.Session.ID

	return json.Unmarshal(response.Session.Config, &sessionPtr.Config)
}
