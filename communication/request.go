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

// RequestEndpoint is special type that describes unique requests endpoint
type RequestEndpoint string

// RequestProducer represents instance which creates requests/responses of specific endpoint
type RequestProducer interface {
	// GetRequestEndpoint returns endpoint where to send requests
	GetRequestEndpoint() RequestEndpoint
	// Produce creates request which will be serialized to endpoint
	Produce() (requestPtr interface{})
	// NewResponse creates struct where responses from endpoint will be serialized
	NewResponse() (responsePtr interface{})
}

// RequestConsumer represents instance which handles requests/responses of specific endpoint
type RequestConsumer interface {
	// GetRequestEndpoint returns endpoint where to receive requests
	GetRequestEndpoint() RequestEndpoint
	// NewRequest creates struct where request from endpoint will be serialized
	NewRequest() (requestPtr interface{})
	// Consume handles requests from endpoint and replies with response
	Consume(requestPtr interface{}) (responsePtr interface{}, err error)
}
