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
)

type Producer struct {
	SignedPromise *SignedPromise
}

// Request structure represents message from service provider to receive new promise from consumer
type Request struct {
	SignedPromise *SignedPromise
}

// Response structure represents service provider response to given session request from consumer
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (p *Producer) GetRequestEndpoint() communication.RequestEndpoint {
	return endpoint
}

func (p *Producer) NewResponse() (responsePtr interface{}) {
	return &Response{}
}

func (p *Producer) Produce() (requestPtr interface{}) {
	return &Request{
		SignedPromise: p.SignedPromise,
	}
}
