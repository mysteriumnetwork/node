package client_local_api

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestHealthCheckReturnsExcpectedResponse(t *testing.T) {

	go Bootstrap(":8000")

	resp := doGet(t, "/healthcheck")

	var jsonMap map[string]interface{}
	expectJsonAndStatus(t, resp, 200, &jsonMap)

	assert.NotEmpty(t, jsonMap["uptime"])
}

func doGet(t *testing.T, path string) *http.Response {

	resp, err := http.Get("http://localhost:8000/" + path)
	assert.Nil(t, err)

	return resp
}

func expectJsonAndStatus(t *testing.T, resp *http.Response, httpStatus int, v interface{}) {
	assert.Equal(t, resp.Header.Get("Content-type"), "application/json")
	assert.Equal(t, resp.StatusCode, httpStatus)

	err := json.NewDecoder(resp.Body).Decode(v)
	assert.Nil(t, err)
}
