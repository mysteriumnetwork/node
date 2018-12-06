/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

	"github.com/mysteriumnetwork/node/communication"
)

type destroyProducer struct {
	SessionID string
}

func (producer *destroyProducer) GetRequestEndpoint() communication.RequestEndpoint {
	return endpointSessionDestroy
}

func (producer *destroyProducer) NewResponse() (responsePtr interface{}) {
	return &DestroyResponse{}
}

func (producer *destroyProducer) Produce() (requestPtr interface{}) {
	return &DestroyRequest{
		SessionID: producer.SessionID,
	}
}

// RequestSessionDestroy requests session destruction and returns response data
func RequestSessionDestroy(sender communication.Sender, sessionID ID) (err error) {
	responsePtr, err := sender.Request(&destroyProducer{
		SessionID: string(sessionID),
	})
	if err != nil {
		return
	}

	response := responsePtr.(*DestroyResponse)
	if !response.Success {
		err = errors.New("Session destroy failed. " + response.Message)
		return
	}

	return
}
