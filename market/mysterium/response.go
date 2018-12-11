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

package mysterium

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

// ParseResponseError checks the respose for correctness and return error if it is invalid
func ParseResponseError(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var error errorResponse

		err := ParseResponseJSON(response, &error)
		if err != nil {
			return err
		}

		return fmt.Errorf("server response invalid: %s (%s) error message: %s",
			response.Status, response.Request.URL, error.Error)
	}
	return nil
}

// ParseResponseJSON parse response to the provided object
func ParseResponseJSON(response *http.Response, dto interface{}) error {
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
