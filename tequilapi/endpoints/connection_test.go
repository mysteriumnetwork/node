package endpoints

import (
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestError404IsReturnedIfThereIsNoActiveConnection(t *testing.T) {

	connEndpoint := connectionEndpoint{}
	req, err := http.NewRequest(http.MethodGet, "/connection", nil)
	assert.Nil(t, err)
	resp := httptest.NewRecorder()

	connEndpoint.Status(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusNotFound, resp.Code)
}
