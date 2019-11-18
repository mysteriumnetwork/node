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
	successfullSessionConfig = json.RawMessage(`{"Param1":"string-param","Param2":123}`)
	successfullSessionID     = ID("session-id")
)

func TestProducer_RequestSessionCreate(t *testing.T) {
	sender := &fakeSender{}
	sessionData, paymentInfo, err := RequestSessionCreate(sender, 123, []byte{}, ConsumerInfo{})
	assert.NoError(t, err)
	assert.Exactly(t, successfullSessionID, sessionData.ID)
	assert.Exactly(t, successfullSessionConfig, sessionData.Config)
	assert.Nil(t, paymentInfo)
}

func TestProducer_SessionAcknowledge(t *testing.T) {
	sender := &fakeSender{}
	err := AcknowledgeSession(sender, string(successfullSessionID))
	assert.NoError(t, err)
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
