package endpoints

import (
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCurrentIPEndpointSucceeds(t *testing.T) {
	endpoint := NewCurrentIPEndpoint(func() (string, error) {
		return "123.123.123.123", nil
	})

	resp := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/irrelevant", nil)
	endpoint.CurrentIP(resp, req, httprouter.Params{})

	assert.JSONEq(
		t,
		`{
			"ip": "123.123.123.123"
		}`,
		resp.Body.String(),
	)
}

func TestCurrentIPEndpointReturnsErrorWhenIPDetectionFails(t *testing.T) {
	endpoint := NewCurrentIPEndpoint(func() (string, error) {
		return "", errors.New("fake error")
	})

	resp := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/irrelevant", nil)
	endpoint.CurrentIP(resp, req, httprouter.Params{})

	// TODO: status code

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message": "fake error"
		}`,
		resp.Body.String(),
	)
}
