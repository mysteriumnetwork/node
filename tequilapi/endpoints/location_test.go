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

package endpoints

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/core/connection"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/ip"
	"github.com/mysterium/node/location"
	"github.com/stretchr/testify/assert"
)

type fakeManagerForLocation struct {
	onStatusReturn connection.ConnectionStatus
}

func (fm *fakeManagerForLocation) Connect(consumerID identity.Identity, providerID identity.Identity) error {
	return nil
}

func (fm *fakeManagerForLocation) Status() connection.ConnectionStatus {
	return fm.onStatusReturn
}

func (fm *fakeManagerForLocation) Disconnect() error {
	return nil
}

func TestAddRoutesForLocationAddsRoutes(t *testing.T) {
	fakeManager := fakeManagerForLocation{}
	fakeManager.onStatusReturn = connection.ConnectionStatus{
		State: connection.Connected,
	}

	router := httprouter.New()

	currentIpResolver := ip.NewFakeResolver("123.123.123.123")
	currentLocationResolver := location.NewResolverFake("current country")
	currentLocationDetector := location.NewDetectorWithLocationResolver(currentIpResolver, currentLocationResolver)

	originalIpResolver := ip.NewFakeResolver("100.100.100.100")
	originalLocationResolver := location.NewResolverFake("original country")
	originalLocationDetector := location.NewDetectorWithLocationResolver(originalIpResolver, originalLocationResolver)
	originalLocationCache := location.NewLocationCache(originalLocationDetector)
	originalLocationCache.RefreshAndGet()

	AddRoutesForLocation(router, &fakeManager, currentLocationDetector, originalLocationCache)

	tests := []struct {
		method         string
		path           string
		body           string
		expectedStatus int
		expectedJSON   string
	}{
		{
			http.MethodGet, "/location", "",
			http.StatusOK,
			`{
				"original": {"ip": "100.100.100.100", "country": "original country"},
				"current":  {"ip": "123.123.123.123", "country": "current country"}
			}`,
		},
	}

	for _, test := range tests {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		router.ServeHTTP(resp, req)
		assert.Equal(t, test.expectedStatus, resp.Code)
		if test.expectedJSON != "" {
			assert.JSONEq(t, test.expectedJSON, resp.Body.String())
		} else {
			assert.Equal(t, "", resp.Body.String())
		}
	}
}

func TestGetLocationWhenConnected(t *testing.T) {
	fakeManager := fakeManager{}
	fakeManager.onStatusReturn = connection.ConnectionStatus{
		State: connection.Connected,
	}

	currentIpResolver := ip.NewFakeResolver("123.123.123.123")
	currentLocationResolver := location.NewResolverFake("current country")
	currentLocationDetector := location.NewDetectorWithLocationResolver(currentIpResolver, currentLocationResolver)

	originalIpResolver := ip.NewFakeResolver("100.100.100.100")
	originalLocationResolver := location.NewResolverFake("original country")
	originalLocationDetector := location.NewDetectorWithLocationResolver(originalIpResolver, originalLocationResolver)
	originalLocationCache := location.NewLocationCache(originalLocationDetector)
	originalLocationCache.RefreshAndGet()

	connEndpoint := NewLocationEndpoint(&fakeManager, currentLocationDetector, originalLocationCache)
	resp := httptest.NewRecorder()

	connEndpoint.GetLocation(resp, nil, nil)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{
			"original": {"ip": "100.100.100.100", "country": "original country"},
			"current":  {"ip": "123.123.123.123", "country": "current country"}
		}`,
		resp.Body.String(),
	)
}

func TestGetLocationWhenNotConnected(t *testing.T) {
	originalIpResolver := ip.NewFakeResolver("100.100.100.100")
	originalLocationResolver := location.NewResolverFake("original country")
	originalLocationDetector := location.NewDetectorWithLocationResolver(originalIpResolver, originalLocationResolver)
	originalLocationCache := location.NewLocationCache(originalLocationDetector)
	originalLocationCache.RefreshAndGet()

	states := []connection.State{
		connection.NotConnected,
		connection.Connecting,
		connection.Disconnecting,
		connection.Reconnecting,
	}

	for _, state := range states {

		fakeManager := fakeManager{}
		fakeManager.onStatusReturn = connection.ConnectionStatus{
			State: state,
		}

		connEndpoint := NewLocationEndpoint(&fakeManager, nil, originalLocationCache)
		resp := httptest.NewRecorder()

		connEndpoint.GetLocation(resp, nil, nil)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.JSONEq(
			t,
			`{
			"original": {"ip": "100.100.100.100", "country": "original country"},
			"current":  {"ip": "100.100.100.100", "country": "original country"}
		}`,
			resp.Body.String(),
		)
	}
}
