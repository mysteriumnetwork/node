package endpoints

import (
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthCheckReturnsExpectedJsonObject(t *testing.T) {

	req := httptest.NewRequest("GET", "/irrelevant", nil)
	resp := httptest.NewRecorder()

	startTime := time.Unix(0, 0)
	requestTime := startTime.Add(time.Minute)

	handlerFunc := HealthCheckEndpointFactory(timeValues([]time.Time{startTime, requestTime}))
	handlerFunc(resp, req)

	assert.JSONEq(
		t,
		`{
			"uptime" : "1m0s"
		}`,
		resp.Body.String())
}

func timeValues(values []time.Time) func() time.Time {
	var currentIndex = 0
	return func() time.Time {
		currentValue := values[currentIndex]
		currentIndex = currentIndex + 1
		return currentValue
	}
}
