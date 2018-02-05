package endpoints

import (
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeStopper struct {
	stopped bool
}

func (fs *fakeStopper) Stop() {
	fs.stopped = true
}

func TestAddRouteForStop(t *testing.T) {
	stopper := fakeStopper{}
	router := httprouter.New()
	AddRouteForStop(router, stopper.Stop)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/stop", strings.NewReader(""))
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusAccepted, resp.Code)

	assert.True(t, stopper.stopped)
}
