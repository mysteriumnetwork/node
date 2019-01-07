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

	"github.com/pkg/errors"
)

// ParseResponseError checks the respose for correctness and return error if it is invalid
func ParseResponseError(response *http.Response) (err error) {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var errorMsg []byte

		if response.Body != nil {
			errorMsg, err = ioutil.ReadAll(response.Body)
			if err != nil {
				return errors.Wrap(err, "server response invalid")
			}
		}

		return fmt.Errorf("server response invalid: %s (%s) error message: %s",
			response.Status, response.Request.URL, errorMsg)
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

	return json.Unmarshal(responseJSON, dto)
}
