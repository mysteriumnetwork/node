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
		EventName:   startupEventName,
		Application: appInfo{Version: "test version"},
	}
)

func TestMORQATransport_SendEvent_HandlesSuccess(t *testing.T) {
	var events metrics.Batch
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, _ := ioutil.ReadAll(request.Body)
		proto.Unmarshal(body, &events)
		response.WriteHeader(http.StatusAccepted)
	}))

	morqa := NewMorqaClient(bindAllAddress, server.URL, 10*time.Millisecond)
	go morqa.Start()
	defer morqa.Stop()

	transport := &morqaTransport{morqaClient: morqa}

	err := transport.SendEvent(eventStartup)
	assert.NoError(t, err)

	assert.Eventually(t, func() bool {
		morqa.sendMetrics()
		return len(events.Events) > 0
	}, time.Second, time.Millisecond)

	assert.Exactly(
		t,
		&metrics.Event{
			IsProvider: false,
			TargetId:   "",
			Metric: &metrics.Event_VersionPayload{
				VersionPayload: &metrics.VersionPayload{
					Version: "test version",
				},
			},
		},
		events.Events[0],
	)
}

func TestMORQAT_sendMetrics_HandlesErrorsWithMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{
			"message": "invalid payload given"
		}`))
	}))

	morqa := NewMorqaClient(bindAllAddress, server.URL, 10*time.Millisecond)
	morqa.addMetric(&metrics.Event{})
	err := morqa.sendMetrics()

	assert.EqualError(t, err, fmt.Sprintf(
		"server response invalid: 400 Bad Request (%s/batch). Possible error: invalid payload given",
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

	morqa := NewMorqaClient(bindAllAddress, server.URL, 10*time.Millisecond)
	morqa.addMetric(&metrics.Event{})
	err := morqa.sendMetrics()

	assert.EqualError(t, err, fmt.Sprintf(
		"server response invalid: 422 Unprocessable Entity (%s/batch). Possible error: validation problems",
		server.URL,
	))
}

func TestMORQA_ProposalsMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{ "connects": [{
			"proposalId": { "providerId": "0x61400b27616f3ce15a86e4cd12c27c7a4d1c545c", "serviceType": "openvpn" },
			"connectCount": { "success": 39, "fail": 1, "timeout": 1 }
		}, {
			"proposalId": { "providerId": "0xb724ba4f646babdebaaad1d1aea6b26df568e8f6", "serviceType": "openvpn" },
			"connectCount": { "success": 1, "fail": 11, "timeout": 11 }
		}, {
			"proposalId": { "providerId": "0x093285d0a05ad5d9a05e0dae1eb69e7437fa02c6", "serviceType": "openvpn" },
			"connectCount": { "success": 0, "fail": 27, "timeout": 0 }
		}] }`))
	}))

	morqa := NewMorqaClient(bindAllAddress, server.URL, 10*time.Millisecond)
	metrics := morqa.ProposalsMetrics()

	assert.Equal(t,
		[]ConnectMetric{
			{
				ProposalID:   ProposalID{ProviderID: "0x61400b27616f3ce15a86e4cd12c27c7a4d1c545c", ServiceType: "openvpn"},
				ConnectCount: ConnectCount{Success: 39, Fail: 1, Timeout: 1},
			},
			{
				ProposalID:   ProposalID{ProviderID: "0xb724ba4f646babdebaaad1d1aea6b26df568e8f6", ServiceType: "openvpn"},
				ConnectCount: ConnectCount{Success: 1, Fail: 11, Timeout: 11},
			},
			{
				ProposalID:   ProposalID{ProviderID: "0x093285d0a05ad5d9a05e0dae1eb69e7437fa02c6", ServiceType: "openvpn"},
				ConnectCount: ConnectCount{Success: 0, Fail: 27, Timeout: 0},
			}},
		metrics,
	)
}
