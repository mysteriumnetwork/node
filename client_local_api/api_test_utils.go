package client_local_api

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

//TODO move this to random port for testing
const serverBindAddress = ":8000"

func startServer() {
	go Bootstrap(serverBindAddress)
}

func doGet(t *testing.T, path string) *http.Response {

	resp, err := http.Get("http://localhost" + serverBindAddress + "/" + path)
	assert.Nil(t, err)
	return resp
}

func expectJsonAndStatus(t *testing.T, resp *http.Response, httpStatus int, v interface{}) {
	assert.Equal(t, "application/json", resp.Header.Get("Content-type"))
	assert.Equal(t, httpStatus, resp.StatusCode)

	err := json.NewDecoder(resp.Body).Decode(v)
	assert.Nil(t, err)
}
