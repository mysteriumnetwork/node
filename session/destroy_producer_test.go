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
	"testing"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/stretchr/testify/assert"
)

var (
	successfulSessionDestroyResponse = &DestroyResponse{
		Success: true,
	}
)

func TestProducer_RequestSessionDestroy(t *testing.T) {
	sender := &fakeSender{}
	sid, _, err := RequestSessionCreate(sender, 123, []byte{}, ConsumerInfo{})
	assert.NoError(t, err)

	destroySender := &fakeDestroySender{}
	err = RequestSessionDestroy(destroySender, sid)
	assert.NoError(t, err)
}

type fakeDestroySender struct {
}

func (sender *fakeDestroySender) Send(producer communication.MessageProducer) error {
	return nil
}

func (sender *fakeDestroySender) Request(producer communication.RequestProducer) (responsePtr interface{}, err error) {
	return successfulSessionDestroyResponse, nil
}
