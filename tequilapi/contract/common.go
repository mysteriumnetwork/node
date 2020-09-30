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

func bindString(ptr *string, str string) *validation.FieldError {
	if str == "" {
		return &validation.FieldError{Code: "required", Message: "Field is required"}
	}

	*ptr = str
	return nil
}

func bindInt(ptr *int, str string) *validation.FieldError {
	if str == "" {
		return &validation.FieldError{Code: "required", Message: "Field is required"}
	}

	value, err := parseInt(str)
	if err != nil {
		return err
	}

	*ptr = *value
	return nil
}

func parseInt(str string) (*int, *validation.FieldError) {
	value, err := strconv.Atoi(str)
	if err != nil {
		return nil, &validation.FieldError{Code: "invalid", Message: err.Error()}
	}

	return &value, nil
}

func bindDate(ptr *strfmt.Date, str string) *validation.FieldError {
	if str == "" {
		return &validation.FieldError{Code: "required", Message: "Field is required"}
	}

	value, err := parseDate(str)
	if err != nil {
		return err
	}

	*ptr = *value
	return nil
}

func parseDate(str string) (*strfmt.Date, *validation.FieldError) {
	value, err := defaultFormats.Parse("date", str)
	if err != nil {
		return nil, &validation.FieldError{Code: "invalid", Message: err.Error()}
	}

	return value.(*strfmt.Date), nil
}
