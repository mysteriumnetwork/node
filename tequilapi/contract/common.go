/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package contract

import (
	"strconv"

	"github.com/go-openapi/strfmt"
	"github.com/mysteriumnetwork/node/tequilapi/validation"
)

var defaultFormats = strfmt.NewFormats()

func parseInt(str string) (*int, *validation.FieldError) {
	value, err := strconv.Atoi(str)
	if err != nil {
		return nil, &validation.FieldError{Code: "invalid", Message: err.Error()}
	}

	return &value, nil
}

func parseDate(str string) (*strfmt.Date, *validation.FieldError) {
	value, err := defaultFormats.Parse("date", str)
	if err != nil {
		return nil, &validation.FieldError{Code: "invalid", Message: err.Error()}
	}

	return value.(*strfmt.Date), nil
}

// ErrorResponse common error response
// swagger:model ErrorResponse
type ErrorResponse struct {
	OriginalError string `json:"original_error"`
	Message       string `json:"message"`
}

// WithErrorResponse factory function
func WithErrorResponse(msg string, err error) ErrorResponse {
	return ErrorResponse{OriginalError: err.Error(), Message: msg}
}

// InvalidRequestError factory function
func InvalidRequestError(err error) ErrorResponse {
	return ErrorResponse{OriginalError: err.Error(), Message: "invalid request"}
}

// InternalError factory function
func InternalError(err error) ErrorResponse {
	return ErrorResponse{OriginalError: err.Error(), Message: "internal error"}
}
