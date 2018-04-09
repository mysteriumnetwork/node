package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

func parseResponseError(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var error errorResponse
		parseResponseJSON(response, &error)
		return fmt.Errorf("server response invalid: %s (%s) error message: %s",
			response.Status, response.Request.URL, error)
	}
	return nil
}

func parseResponseJSON(response *http.Response, dto interface{}) error {
	if response.Body == nil {
		return nil
	}

	responseJSON, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(responseJSON, dto)
	if err != nil {
		return err
	}

	return nil
}
