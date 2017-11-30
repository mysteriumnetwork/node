package client_local_api

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	startServer() //start a single server before all tests
	os.Exit(m.Run())
}

func TestHealthCheckReturnsExcpectedResponse(t *testing.T) {

	resp := doGet(t, "/healthcheck")

	var jsonMap map[string]interface{}
	expectJsonAndStatus(t, resp, 200, &jsonMap)

	assert.NotEmpty(t, jsonMap["uptime"])
}
