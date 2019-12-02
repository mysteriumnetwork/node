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

package mysterium

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/stretchr/testify/assert"
)

const bindAllAddress = "0.0.0.0"

func TestHttpTransportDoesntBlockForeverIfServerFailsToSendAnyResponse(t *testing.T) {

	address, err := createHTTPServer(func(writer http.ResponseWriter, request *http.Request) {
		select {} //infinite loop with no response to client
	})
	assert.NoError(t, err)

	httpClient := requests.NewHTTPClient(bindAllAddress, 50*time.Millisecond)
	httpClient.StopTransportRetries()
	req, err := http.NewRequest(http.MethodGet, "http://"+address+"/", nil)
	assert.NoError(t, err)

	completed := make(chan error)
	go func() {
		_, err := httpClient.Do(req)
		completed <- err
	}()

	select {
	case err := <-completed:
		netError, ok := err.(net.Error)
		assert.True(t, ok)
		assert.True(t, netError.Timeout())
	case <-time.After(1200 * time.Millisecond):
		assert.Fail(t, "failed request expected")
	}
}

func TestProposalsReturnsPreviousProposalsWhenEtagMatches(t *testing.T) {
	sentClientEtag := make(chan string, 1)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sentClientEtag <- r.Header.Get("If-None-Match")
		w.WriteHeader(http.StatusNotModified)
	}))
	defer s.Close()

	client := NewClient(requests.NewHTTPClient("0.0.0.0", requests.DefaultTimeout), s.URL)
	client.latestProposals = []market.ServiceProposal{
		{ID: 1},
		{ID: 2},
	}
	client.latestProposalsEtag = "etag1"

	proposals, err := client.Proposals()

	assert.NoError(t, err)
	assert.Len(t, proposals, 2)
	assert.Equal(t, "etag1", <-sentClientEtag)
}

func TestProposalsOverrideLatestProposalsWhenEtagDoNotMatch(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := []byte(`{"proposals": []}`)
		w.Header().Add("Etag", "etag1")
		w.Write(res)
	}))
	defer s.Close()

	client := NewClient(requests.NewHTTPClient("0.0.0.0", requests.DefaultTimeout), s.URL)
	client.latestProposals = []market.ServiceProposal{
		{ID: 1},
		{ID: 2},
	}
	proposals, err := client.Proposals()

	assert.NoError(t, err)
	assert.Len(t, proposals, 0)
	assert.Equal(t, "etag1", client.latestProposalsEtag)
}

func createHTTPServer(handlerFunc http.HandlerFunc) (address string, err error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return
	}

	go http.Serve(listener, handlerFunc)
	return listener.Addr().String(), nil
}
