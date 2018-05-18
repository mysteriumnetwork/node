/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package utils

import (
	"errors"
	"github.com/mysterium/node/tequilapi/validation"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteAsJSONReturnsExpectedResponse(t *testing.T) {

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

	SendErrorBody(resp, errorMessage{"error_message"}, http.StatusInternalServerError)

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
