package utils

import (
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/mysterium/node/tequilapi/validation"
	"net/http"
)

/*
WriteAsJson takes value as the first argument and handles json marshaling with returning appropriate errors if needed,
also enforces application/json and charset response headers
*/
func WriteAsJson(v interface{}, writer http.ResponseWriter) {

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

func SendError(writer http.ResponseWriter, err error, httpCode int) {
	SendErrorMessage(writer, &errorMessage{fmt.Sprint(err)}, httpCode)
}

func SendErrorMessage(writer http.ResponseWriter, message interface{}, httpCode int) {
	writer.WriteHeader(httpCode)
	WriteAsJson(message, writer)
}

type validationErrorMessage struct {
	errorMessage
	ValidationErrors *validation.FieldErrorMap `json:"errors"`
}

func SendValidationErrorMessage(resp http.ResponseWriter, errorMap *validation.FieldErrorMap) {
	errorResponse := errorMessage{Message: "validation_error"}

	SendErrorMessage(resp, &validationErrorMessage{errorResponse, errorMap}, http.StatusUnprocessableEntity)
}
