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

package communication

// MessageEndpoint is special type that describes unique message endpoint
type MessageEndpoint string

// MessageProducer represents instance which creates messages to specific endpoint
type MessageProducer interface {
	// GetMessageEndpoint returns endpoint where to send messages
	GetMessageEndpoint() MessageEndpoint
	// Produce creates message which will be serialized to endpoint
	Produce() (messagePtr interface{})
}

// MessageConsumer represents instance which handles messages of specific endpoint
type MessageConsumer interface {
	// GetMessageEndpoint returns endpoint where to receive messages
	GetMessageEndpoint() MessageEndpoint
	// NewMessage creates struct where message from endpoint will be serialized
	NewMessage() (messagePtr interface{})
	// Consume handles messages from endpoint
	Consume(messagePtr interface{}) error
}
