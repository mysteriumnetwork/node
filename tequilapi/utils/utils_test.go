package utils

import (
	"errors"
	"github.com/mysterium/node/tequilapi/validation"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteAsJsonReturnsExpectedResponse(t *testing.T) {

	respRecorder := httptest.NewRecorder()

	type TestStruct struct {
		IntField    int
		StringField string `json:"renamed"`
	}

	WriteAsJSON(TestStruct{1, "abc"}, respRecorder)

	result := respRecorder.Result()

	assert.Equal(t, "application/json", result.Header.Get("Content-type"))
	assert.JSONEq(
		t,
		`{
			"IntField" : 1,
			"renamed" : "abc"
		}`,
		respRecorder.Body.String())
}

func TestSendErrorRendersErrorMessage(t *testing.T) {
	resp := httptest.NewRecorder()

	SendError(resp, errors.New("custom_error"), http.StatusInternalServerError)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message" : "custom_error"
		}`,
		resp.Body.String())
}

func TestSendErrorMessageRendersErrorMessage(t *testing.T) {
	resp := httptest.NewRecorder()

	SendErrorMessage(resp, errorMessage{"error_message"}, http.StatusInternalServerError)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message" : "error_message"
		}`,
		resp.Body.String())

}

func TestSendValidationErrorMessageRendersErrorMessage(t *testing.T) {
	resp := httptest.NewRecorder()

	errorMap := validation.NewErrorMap()
	errorMap.ForField("email").AddError("required", "field required")

	SendValidationErrorMessage(resp, errorMap)

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message" : "validation_error" ,
			"errors" : {
				"email" : [
					{ "code" : "required" , "message" : "field required"}
				]
			}
		}`,
		resp.Body.String())

}
