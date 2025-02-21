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

package control

import (
	"github.com/mysteriumnetwork/node/communication"
)

var _ communication.MessageConsumer = (*consumer)(nil)

// consumer represents the pub/sub consumer of control messages
type consumer struct {
	callback func(controlMessage) error
	topic    communication.MessageEndpoint
}

// GetMessageEndpoint returns endpoint where to receive messages
func (c *consumer) GetMessageEndpoint() (communication.MessageEndpoint, error) {
	return c.topic, nil
}

// NewMessage creates struct where message from endpoint will be serialized
func (c *consumer) NewMessage() (messagePtr interface{}) {
	return &controlMessage{}
}

// Consume handles messages from endpoint
func (c *consumer) Consume(messagePtr interface{}) error {
	return c.callback(*messagePtr.(*controlMessage))
}
