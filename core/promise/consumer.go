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

type consumer struct{}

func (c *consumer) GetRequestEndpoint() communication.RequestEndpoint {
	return endpoint
}

func (c *consumer) NewRequest() (requestPtr interface{}) {
	return &Request{}
}

func (c *consumer) Consume(requestPtr interface{}) (response interface{}, err error) {
	// request := requestPtr.(*Request)

	// TODO there should be some validation of the received proposal and storing it somewhere for the server needs.
	// TODO signature validation of the promise should be here too.

	// if request.SignedPromise.IssuerSignature == valid {
	// 		return &Response{
	// 			Success: false,
	// 			Message: fmt.Sprintf("Bas signature: %s", request.SignedPromise.IssuerSignature),
	// 		}, fmt.Errorf("Bad promise signature")
	// }

	return &Response{Success: true, Message: "Promise accepted"}, nil
}
