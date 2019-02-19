/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mysteriumnetwork/node/tequilapi/validation"
)

/*
WriteAsJSON takes value as the first argument and handles json marshaling with returning appropriate errors if needed,
also enforces application/json and charset response headers
*/
func WriteAsJSON(v interface{}, writer http.ResponseWriter) {

	writer.Header().Set("Content-type", "application/json; charset=utf-8")

	writeErr := json.NewEncoder(writer).Encode(v)
	if writeErr != nil {
		http.Error(writer, "Http response write error", http.StatusInternalServerError)
	}
}

// swagger:model ErrorMessageDTO
type errorMessage struct {
	// example: error message
	Message string `json:"message"`
}

// SendError generates error response for error
func SendError(writer http.ResponseWriter, err error, httpCode int) {
	SendErrorMessage(writer, fmt.Sprint(err), httpCode)
}

// SendErrorMessage generates error response with custom json message
func SendErrorMessage(writer http.ResponseWriter, message string, httpCode int) {
	SendErrorBody(writer, &errorMessage{message}, httpCode)
}

// SendErrorBody generates error response with custom body
func SendErrorBody(writer http.ResponseWriter, message interface{}, httpCode int) {
	writer.WriteHeader(httpCode)
	WriteAsJSON(message, writer)
}

// swagger:model ValidationErrorDTO
type validationErrorMessage struct {
	errorMessage
	ValidationErrors *validation.FieldErrorMap `json:"errors"`
}

// SendValidationErrorMessage generates error response for validation errors
func SendValidationErrorMessage(resp http.ResponseWriter, errorMap *validation.FieldErrorMap) {
	errorResponse := errorMessage{Message: "validation_error"}

	SendErrorBody(resp, &validationErrorMessage{errorResponse, errorMap}, http.StatusUnprocessableEntity)
}
