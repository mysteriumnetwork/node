package tequilapi

import (
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

type handleFunctionTestStruct struct {
	called bool
}

func (hfts *handleFunctionTestStruct) httprouterHandle(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	hfts.called = true
}

func TestHttpRouterHandlesRequests(t *testing.T) {
	ts := handleFunctionTestStruct{false}

	router := NewApiEndpoints()
	router.GET("/testhandler", ts.httprouterHandle)

	req, err := http.NewRequest("GET", "/testhandler", nil)
	assert.Nil(t, err)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)
	assert.Equal(t, true, ts.called)
}
