package endpoints

import (
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/version"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthCheckReturnsExpectedJsonObject(t *testing.T) {

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
	mockTimer.current += 1
	return value
}
