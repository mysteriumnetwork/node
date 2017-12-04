package tequilapi

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

type TestClient interface {
	Get(path string) *http.Response
}

type testClient struct {
	t       *testing.T
	baseUrl string
}

func NewTestClient(t *testing.T, port int) TestClient {
	return &testClient{
		t,
		fmt.Sprintf("http://127.0.0.1:%d", port),
	}
}

func (tc *testClient) Get(path string) *http.Response {
	resp, err := http.Get(tc.baseUrl + path)
	if err != nil {
		assert.FailNow(tc.t, "Uh oh catched error: ", err.Error())
	}
	return resp
}

func expectJsonAndStatus(t *testing.T, resp *http.Response, httpStatus int, v interface{}) {
	assert.Equal(t, "application/json", resp.Header.Get("Content-type"))
	assert.Equal(t, httpStatus, resp.StatusCode)

	err := json.NewDecoder(resp.Body).Decode(v)
	assert.Nil(t, err)
}
