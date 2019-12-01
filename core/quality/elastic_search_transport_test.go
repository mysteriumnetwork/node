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
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/requests"
	"github.com/stretchr/testify/assert"
)

const bindAllAddress = "0.0.0.0"

func TestElasticSearchTransport_SendEvent_Success(t *testing.T) {
	invoked := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buffer := new(bytes.Buffer)
		buffer.ReadFrom(r.Body)
		body := buffer.String()

		assert.JSONEq(t, `{
			"application": {
				"name": "test app",
				"version": "test version",
				"os": "`+runtime.GOOS+`",
				"arch": "`+runtime.GOARCH+`"
			},
			"createdAt": 0,
			"eventName": "",
			"context": null
		}`, body)

		fmt.Fprint(w, "ok")
		invoked = true
	}))

	app := appInfo{Name: "test app", Version: "test version", OS: runtime.GOOS, Arch: runtime.GOARCH}
	event := Event{Application: app}

	transport := NewElasticSearchTransport(requests.NewHTTPClient(bindAllAddress, requests.DefaultTimeout), server.URL, time.Second)
	transport.SendEvent(event)

	assert.True(t, invoked)
}

func TestElasticSearchTransport_SendEvent_WithUnexpectedStatus(t *testing.T) {
	invoked := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "something not cool happened")
		invoked = true
	}))

	transport := NewElasticSearchTransport(requests.NewHTTPClient(bindAllAddress, requests.DefaultTimeout), server.URL, time.Second)
	transport.SendEvent(Event{})

	assert.True(t, invoked)
}

func TestElasticSearchTransport_SendEvent_WithUnexpectedResponse(t *testing.T) {
	invoked := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "bad")
		invoked = true
	}))

	transport := NewElasticSearchTransport(requests.NewHTTPClient(bindAllAddress, requests.DefaultTimeout), server.URL, time.Second)
	transport.SendEvent(Event{})

	assert.True(t, invoked)
}
