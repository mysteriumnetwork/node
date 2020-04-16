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

package endpoints

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type locationResolverMock struct {
	ip       string
	ipOrigin string
}

func (r *locationResolverMock) DetectLocation() (location.Location, error) {
	loc := location.Location{
		ASN:       62179,
		City:      "Vilnius",
		Continent: "EU",
		Country:   "LT",
		IP:        r.ip,
		ISP:       "Telia Lietuva, AB",
		NodeType:  "residential",
	}

	return loc, nil
}

func (r *locationResolverMock) GetOrigin() (location.Location, error) {
	loc := location.Location{
		ASN:       62179,
		City:      "Vilnius",
		Continent: "EU",
		Country:   "LT",
		IP:        r.ipOrigin,
		ISP:       "Telia Lietuva, AB",
		NodeType:  "residential",
	}

	return loc, nil
}

func TestAddRoutesForConnectionLocationAddsRoutes(t *testing.T) {
	router := httprouter.New()

	locationResolver := &locationResolverMock{ip: "1.2.3.4", ipOrigin: "1.2.3.1"}
	AddRoutesForConnectionLocation(
		router,
		ip.NewResolverMock("123.123.123.123"),
		locationResolver,
		locationResolver,
	)

	tests := []struct {
		method         string
		path           string
		body           string
		expectedStatus int
		expectedJSON   string
	}{
		{
			http.MethodGet, "/connection/ip", "",
			http.StatusOK, `{"ip": "123.123.123.123"}`,
		},
		{
			http.MethodGet, "/connection/location", "",
			http.StatusOK,
			`{
				"asn": 62179,
				"city": "Vilnius",
				"continent": "EU",
				"country": "LT",
				"ip": "1.2.3.4",
				"isp": "Telia Lietuva, AB",
				"user_type": "residential",
				"node_type": "residential"
			}`,
		},
		{
			http.MethodGet, "/location", "",
			http.StatusOK,
			`{
				"asn": 62179,
				"city": "Vilnius",
				"continent": "EU",
				"country": "LT",
				"ip": "1.2.3.1",
				"isp": "Telia Lietuva, AB",
				"user_type": "residential",
				"node_type": "residential"
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

func TestGetIPEndpointSucceeds(t *testing.T) {
	ipResolver := ip.NewResolverMock("123.123.123.123")
	endpoint := NewConnectionLocationEndpoint(ipResolver, nil, nil)
	resp := httptest.NewRecorder()

	endpoint.GetConnectionIP(resp, nil, nil)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{
			"ip": "123.123.123.123"
		}`,
		resp.Body.String(),
	)
}

func TestGetIPEndpointReturnsErrorWhenIPDetectionFails(t *testing.T) {
	ipResolver := ip.NewResolverMockFailing(errors.New("fake error"))
	endpoint := NewConnectionLocationEndpoint(ipResolver, nil, nil)
	resp := httptest.NewRecorder()

	endpoint.GetConnectionIP(resp, nil, nil)

	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message": "fake error"
		}`,
		resp.Body.String(),
	)
}
