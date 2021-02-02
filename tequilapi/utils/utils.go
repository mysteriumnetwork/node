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

	"github.com/rs/zerolog/log"
)

// WriteAsJSON writes a given value `v` to a given http.ResponseWritter
// forcing `content-type application/json`. Optional httpCode parameter
// can be given to also write a specific status code.
func WriteAsJSON(v interface{}, writer http.ResponseWriter, httpCode ...int) {
	writer.Header().Set("Content-type", "application/json; charset=utf-8")

	blob, err := json.Marshal(v)
	if err != nil {
		http.Error(writer, "Http response write error", http.StatusInternalServerError)
		return
	}

	if len(httpCode) > 0 {
		writer.WriteHeader(httpCode[0])
	}

	if _, err := writer.Write(blob); err != nil {
		log.Error().Err(err).Msg("Writing response body failed")
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
	WriteAsJSON(&errorMessage{message}, writer, httpCode)
}

// swagger:model ValidationErrorDTO
type validationErrorMessage struct {
	errorMessage
	ValidationErrors *validation.FieldErrorMap `json:"errors"`
}

// SendValidationErrorMessage generates error response for validation errors
func SendValidationErrorMessage(resp http.ResponseWriter, errorMap *validation.FieldErrorMap) {
	errorResponse := errorMessage{Message: "validation_error"}
	WriteAsJSON(&validationErrorMessage{errorResponse, errorMap}, resp, http.StatusUnprocessableEntity)
}
