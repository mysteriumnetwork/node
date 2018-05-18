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

package endpoints

import (
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/version"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthCheckReturnsExpectedJSONObject(t *testing.T) {

	req := httptest.NewRequest("GET", "/irrelevant", nil)
	resp := httptest.NewRecorder()

	tick1 := time.Unix(0, 0)
	tick2 := tick1.Add(time.Minute)

	handlerFunc := HealthCheckEndpointFactory(
		newMockTimer([]time.Time{tick1, tick2}).Now,
		func() int { return 1 },
		&version.Info{
			Branch:      "some",
			Commit:      "abc123",
			BuildNumber: "travis build #",
		},
	).HealthCheck
	handlerFunc(resp, req, httprouter.Params{})

	assert.JSONEq(
		t,
		`{
            "uptime" : "1m0s",
            "process" : 1,
            "version" : {
                "branch": "some",
                "commit": "abc123",
                "buildNumber": "travis build #"
            }
        }`,
		resp.Body.String())
}

type mockTimer struct {
	values  []time.Time
	current int
}

func newMockTimer(values []time.Time) *mockTimer {
	return &mockTimer{
		values,
		0,
	}
}

func (mockTimer *mockTimer) Now() time.Time {
	value := mockTimer.values[mockTimer.current%len(mockTimer.values)]
	mockTimer.current++
	return value
}
