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

package communication

// RequestEndpoint is special type that describes unique message endpoint
type RequestEndpoint string

// RequestProducer represents instance which creates requests/responses of specific endpoint
type RequestProducer interface {
	GetRequestEndpoint() RequestEndpoint
	NewResponse() (responsePtr interface{})
	Produce() (requestPtr interface{})
}

// RequestConsumer represents instance which handles requests/responses of specific endpoint
type RequestConsumer interface {
	GetRequestEndpoint() RequestEndpoint
	NewRequest() (messagePtr interface{})
	Consume(requestPtr interface{}) (responsePtr interface{}, err error)
}
