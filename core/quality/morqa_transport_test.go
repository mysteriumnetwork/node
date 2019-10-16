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

package quality

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/mysteriumnetwork/metrics"
	"github.com/stretchr/testify/assert"
)

var (
	eventStartup = Event{
		EventName:   eventNameStartup,
		Application: appInfo{Version: "test version"},
	}
)

func TestMORQATransport_SendEvent_HandlesSuccess(t *testing.T) {
	var lastEvent metrics.Event
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, _ := ioutil.ReadAll(request.Body)
		proto.Unmarshal(body, &lastEvent)
		response.WriteHeader(http.StatusAccepted)
	}))

	transport := &morqaTransport{morqaClient: NewMorqaClient(server.URL, 10*time.Millisecond)}
	err := transport.SendEvent(eventStartup)

	assert.NoError(t, err)
	assert.Exactly(
		t,
		metrics.Event{
			IsProvider: false,
			TargetId:   "",
			Metric: &metrics.Event_VersionPayload{
				VersionPayload: &metrics.VersionPayload{
					Version: "test version",
				},
			},
		},
		lastEvent,
	)
}

func TestMORQATransport_SendEvent_HandlesErrorsWithMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{
			"message": "invalid payload given"
		}`))
	}))

	transport := &morqaTransport{morqaClient: NewMorqaClient(server.URL, 10*time.Millisecond)}
	err := transport.SendEvent(eventStartup)

	assert.EqualError(t, err, fmt.Sprintf(
		"server response invalid: 400 Bad Request (%s/metrics). Possible error: invalid payload given",
		server.URL,
	))
}

func TestMORQATransport_SendEvent_HandlesValidationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{
			"message": "validation problems",
			"errors": {
				"field": [ {"code": "required", "message": "Field is required"} ]
			}
		}`))
	}))

	transport := &morqaTransport{morqaClient: NewMorqaClient(server.URL, 10*time.Millisecond)}
	err := transport.SendEvent(eventStartup)

	assert.EqualError(t, err, fmt.Sprintf(
		"server response invalid: 422 Unprocessable Entity (%s/metrics). Possible error: validation problems",
		server.URL,
	))
}

func TestMORQATransport_SendEvent_HandlesFatalErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{}`))
	}))

	transport := &morqaTransport{morqaClient: NewMorqaClient(server.URL, 10*time.Millisecond)}
	err := transport.SendEvent(eventStartup)

	assert.EqualError(t, err, fmt.Sprintf(
		"POST %s/metrics giving up after 11 attempts",
		server.URL,
	))
}
