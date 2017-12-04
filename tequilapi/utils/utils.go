package utils

import (
	"encoding/json"
	"net/http"
)

/*
WriteAsJson takes value as the first argument and handles json marshaling with returning appropriate errors if needed,
also enforces application/json and charset response headers
*/
func WriteAsJson(v interface{}, writer http.ResponseWriter) {

	serialized, err := json.Marshal(v)
	if err != nil {
		http.Error(writer, "Unable to serialize response body", http.StatusInternalServerError)
		return
	}
	writer.Header().Add("Content-type", "application/json")
	writer.Header().Add("Content-type", "charset=utf-8")
	_, writeErr := writer.Write(serialized)
	if writeErr != nil {
		http.Error(writer, "Http response write error", http.StatusInternalServerError)
	}
}
