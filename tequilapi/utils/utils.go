package utils

import (
	"encoding/json"
	"fmt"
	"github.com/mysterium/node/tequilapi/validation"
	"net/http"
)

/*
WriteAsJSON takes value as the first argument and handles json marshaling with returning appropriate errors if needed,
also enforces application/json and charset response headers
*/
func WriteAsJSON(v interface{}, writer http.ResponseWriter) {

	writer.Header().Add("Content-type", "application/json")
	writer.Header().Add("Content-type", "charset=utf-8")

	writeErr := json.NewEncoder(writer).Encode(v)
	if writeErr != nil {
		http.Error(writer, "Http response write error", http.StatusInternalServerError)
	}
}

type errorMessage struct {
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

type validationErrorMessage struct {
	errorMessage
	ValidationErrors *validation.FieldErrorMap `json:"errors"`
}

// SendValidationErrorMessage generates error response for validation errors
func SendValidationErrorMessage(resp http.ResponseWriter, errorMap *validation.FieldErrorMap) {
	errorResponse := errorMessage{Message: "validation_error"}

	SendErrorBody(resp, &validationErrorMessage{errorResponse, errorMap}, http.StatusUnprocessableEntity)
}
