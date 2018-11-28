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

package promise

import (
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/service_discovery/dto"
)

var (
	responseInvalidPromise = Response{Success: false, Message: "Invalid Promise"}
	responseInternalError  = Response{Success: false, Message: "Internal Error"}
)

// Storer alows storing of data by topic
type Storer interface {
	Store(issuer string, data interface{}) error
}

// Consumer process promise-requests
type Consumer struct {
	proposal dto.ServiceProposal
	balance  identity.Balance
	storage  Storer
}

// NewConsumer creates new instance of the promise consumer
func NewConsumer(proposal dto.ServiceProposal, balance identity.Balance, storage Storer) *Consumer {
	return &Consumer{
		proposal: proposal,
		balance:  balance,
		storage:  storage,
	}
}

// GetRequestEndpoint returns endpoint where to receive requests
func (c *Consumer) GetRequestEndpoint() communication.RequestEndpoint {
	return endpoint
}

// NewRequest creates struct where request from endpoint will be serialized
func (c *Consumer) NewRequest() (requestPtr interface{}) {
	return &Request{}
}

// Consume handles requests from endpoint and replies with response
func (c *Consumer) Consume(requestPtr interface{}) (response interface{}, err error) {
	request, ok := requestPtr.(*Request)
	if !ok {
		return responseInvalidPromise, errUnsupportedRequest
	}

	if err := request.SignedPromise.Validate(c.proposal, c.balance); err != nil {
		return responseInvalidPromise, err
	}

	if err := c.storage.Store(request.SignedPromise.Promise.IssuerID, &request.SignedPromise.Promise); err != nil {
		return responseInternalError, err
	}

	return &Response{Success: true}, nil
}
