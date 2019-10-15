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

package connectivity

import (
	"github.com/mysteriumnetwork/node/communication"
)

const endpointConnectivityStatus = communication.MessageEndpoint("session-connectivity-status")

// StatusCode is a connectivity status.
type StatusCode uint32

const (
	// StatusConnectionOk indicates that session, payments, ip change are all working.
	StatusConnectionOk StatusCode = 1000

	// StatusSessionEstablishmentFailed indicates that session is failed to establish.
	StatusSessionEstablishmentFailed StatusCode = 2000

	// StatusSessionPaymentsFailed indicates that session payments failed.
	StatusSessionPaymentsFailed StatusCode = 2001

	// StatusSessionIPNotChanged indicates that session is established but ip is not changed.
	StatusSessionIPNotChanged StatusCode = 2002

	// StatusConnectionFailed indicates unknown session connection error.
	StatusConnectionFailed StatusCode = 2003
)

// StatusMessage is a contract for message broker.
type StatusMessage struct {
	SessionID  string     `json:"sessionID"`
	StatusCode StatusCode `json:"statusCode"`
	Message    string     `json:"message"`
}

// Producer boilerplate.
type statusProducer struct {
	message *StatusMessage
}

func (p *statusProducer) GetMessageEndpoint() communication.MessageEndpoint {
	return endpointConnectivityStatus
}

func (p *statusProducer) Produce() (messagePtr interface{}) {
	return p.message
}

// Consumer boilerplate.
type statusConsumer struct {
	callback func(msg *StatusMessage)
}

func (c *statusConsumer) GetMessageEndpoint() communication.MessageEndpoint {
	return endpointConnectivityStatus
}

func (c *statusConsumer) NewMessage() (messagePtr interface{}) {
	return &StatusMessage{}
}

func (c *statusConsumer) Consume(messagePtr interface{}) error {
	msg := messagePtr.(*StatusMessage)
	c.callback(msg)
	return nil
}
