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
		ProposalId:   producer.ProposalID,
		Config:       producer.Config,
		ConsumerInfo: producer.ConsumerInfo,
	}
}

// RequestSessionCreate requests session creation and returns session DTO
func RequestSessionCreate(sender communication.Sender, proposalID int, config interface{}, ci ConsumerInfo) (sessionID ID, sessionConfig json.RawMessage, err error) {
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
		err = errors.New("Session create failed. " + response.Message)
		return
	}

	sessionID = response.Session.ID
	sessionConfig = response.Session.Config
	return
}
