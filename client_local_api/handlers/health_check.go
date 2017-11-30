package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

var startupTime = time.Now()

type healthCheckData struct {
	Uptime string `json:"uptime"`
}

var HealthCheckHandler http.HandlerFunc = func(writer http.ResponseWriter, request *http.Request) {
	status := healthCheckData{
		Uptime: time.Now().Sub(startupTime).String(),
	}
	writeAsJson(status, writer)
}

func writeAsJson(v interface{}, writer http.ResponseWriter) {

	serialized, err := json.Marshal(v)
	if err != nil {
		http.Error(writer, "Unable to serialize response body", http.StatusInternalServerError)
		return
	}
	writer.Header().Add("Content-type", "application/json")
	writer.Header().Add("Content-type", "charset=utf-8")
	_, writeErr := writer.Write(serialized)
	if writeErr != nil {
		http.Error(writer, "Unable write to writer (funny :D)", http.StatusInternalServerError)
	}
}
