/*
 * Copyright (C) 2018 The Mysterium Network Authors
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
	"errors"
	"github.com/mysterium/node/communication"
)

type SessionCreateProducer struct {
	ProposalId int
}

func (producer *SessionCreateProducer) GetRequestEndpoint() communication.RequestEndpoint {
	return endpointSessionCreate
}

func (producer *SessionCreateProducer) NewResponse() (responsePtr interface{}) {
	var response SessionCreateResponse
	return &response
}

func (producer *SessionCreateProducer) Produce() (requestPtr interface{}) {
	return &SessionCreateRequest{
		ProposalId: producer.ProposalId,
	}
}

func RequestSessionCreate(sender communication.Sender, proposalId int) (*SessionDto, error) {
	responsePtr, err := sender.Request(&SessionCreateProducer{
		ProposalId: proposalId,
	})
	response := responsePtr.(*SessionCreateResponse)

	if err != nil || !response.Success {
		return nil, errors.New("SessionDto create failed. " + response.Message)
	}

	return &response.Session, nil
}
