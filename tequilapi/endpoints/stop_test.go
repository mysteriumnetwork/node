package endpoints

import (
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeStopper struct {
	stopAllowed chan struct{}
	stopped     chan struct{}
}

func (fs *fakeStopper) AllowStop() {
	fs.stopAllowed <- struct{}{}
}

func (fs *fakeStopper) Stop() {
	<-fs.stopAllowed
	fs.stopped <- struct{}{}
}

func TestAddRouteForStop(t *testing.T) {
	stopper := fakeStopper{
		stopAllowed: make(chan struct{}),
		stopped:     make(chan struct{}),
	}
	router := httprouter.New()
	AddRouteForStop(router, stopper.Stop, 0)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/stop", strings.NewReader(""))
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusAccepted, resp.Code)
	assert.Equal(t, 0, len(stopper.stopped))

	stopper.AllowStop()

	select {
	case <-stopper.stopped:
	case <-time.After(time.Second):
		t.Error("Stopper was not executed")
	}
}
