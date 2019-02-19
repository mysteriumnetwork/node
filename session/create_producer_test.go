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
	"encoding/json"
	"testing"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/stretchr/testify/assert"
)

var (
	succesfullSessionConfig   = json.RawMessage(`{"Param1":"string-param","Param2":123}`)
	succesfullSessionID       = ID("session-id")
	successfulSessionResponse = &CreateResponse{
		Success: true,
		Message: "Everything is great!",
		Session: SessionDto{
			ID:     succesfullSessionID,
			Config: succesfullSessionConfig,
		},
	}
)

type fakeSessionConfig struct {
	Param1 string
	Param2 int
}

func TestProducer_RequestSessionCreate(t *testing.T) {
	sender := &fakeSender{}
	sid, config, err := RequestSessionCreate(sender, 123, []byte{}, ConsumerInfo{})
	assert.NoError(t, err)
	assert.Exactly(t, succesfullSessionID, sid)
	assert.Exactly(t, succesfullSessionConfig, config)
}

type fakeSender struct {
	lastRequest communication.RequestProducer
}

func (sender *fakeSender) Send(producer communication.MessageProducer) error {
	return nil
}

func (sender *fakeSender) Request(producer communication.RequestProducer) (responsePtr interface{}, err error) {
	sender.lastRequest = producer
	return &CreateResponse{
		Success: true,
		Message: "Everything is great!",
		Session: SessionDto{
			ID:     "session-id",
			Config: []byte(`{"Param1":"string-param","Param2":123}`),
		},
	}, nil
}
