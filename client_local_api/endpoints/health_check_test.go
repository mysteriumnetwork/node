package endpoints

import (
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthCheckReturnsExpectedJsonObject(t *testing.T) {

	startupTime = time.Now()
	currentTime = func() time.Time { return startupTime.Add(time.Minute) }

	req := httptest.NewRequest("GET", "/irrelevant", nil)
	resp := httptest.NewRecorder()

	HealthCheckHandler(resp, req)

	assert.JSONEq(t, `
	{
		"uptime" : "1m0s"

	}
	`, resp.Body.String())

}
