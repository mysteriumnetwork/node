package tequilapi

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCorsHeadersAreAppliedToResponse(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/not-important", nil)
	assert.NoError(t, err)

	respRecorder := httptest.NewRecorder()

	mock := &mockedHttpHandler{}

	ApplyCors(mock).ServeHTTP(respRecorder, req)

	assert.NotEmpty(t, respRecorder.Header().Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, respRecorder.Header().Get("Access-Control-Allow-Methods"))
	assert.True(t, mock.wasCalled)
}

type mockedHttpHandler struct {
	wasCalled bool
}

func (mock *mockedHttpHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	mock.wasCalled = true
}
