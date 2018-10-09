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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRequest(t *testing.T) {
	consumer := Consumer{}

	assert.Equal(t, &Request{}, consumer.NewRequest())
}

func TestConsume(t *testing.T) {
	// TODO this test is just placeholder, it should be changed when real consumer will be implemented.
	consumer := Consumer{}

	signedPromise := &SignedPromise{Promise: Promise{}, IssuerSignature: "ProducerSignature"}

	response, err := consumer.Consume(&Request{signedPromise})
	assert.NoError(t, err)
	assert.Equal(t, &Response{Success: true, Message: "Promise accepted"}, response)
}
