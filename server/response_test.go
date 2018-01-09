package server

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestHttpErrorIsReportedAsErrorReturnValue(t *testing.T) {
	req, err := newGetRequest("path", nil)
	assert.NoError(t, err)

	response := &http.Response{
		StatusCode: 400,
		Request:    req,
	}
	err = parseResponseError(response)
	assert.Error(t, err)
}
