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
	"time"

	"github.com/mysteriumnetwork/node/tequilapi/validation"
)

func parseString(str string, errs *validation.FieldErrorList) string {
	if str == "" {
		errs.AddError("required", "Field is required")
		return ""
	}

	return str
}

func parseStringOptional(str string, _ *validation.FieldErrorList) *string {
	if str == "" {
		return nil
	}

	return &str
}

func parseInt(str string, errs *validation.FieldErrorList) int {
	if str == "" {
		errs.AddError("required", "Field is required")
		return 0
	}

	value, err := strconv.Atoi(str)
	if err != nil {
		errs.AddError("invalid", err.Error())
		return 0
	}

	return value
}

func parseIntOptional(str string, errs *validation.FieldErrorList) *int {
	if str == "" {
		return nil
	}

	value, err := strconv.Atoi(str)
	if err != nil {
		errs.AddError("invalid", err.Error())
		return nil
	}

	return &value
}

func parseDate(str string, errs *validation.FieldErrorList) time.Time {
	if str == "" {
		errs.AddError("required", "Field is required")
		return time.Time{}
	}

	value, err := time.Parse(time.RFC3339, str)
	if err != nil {
		errs.AddError("invalid", err.Error())
		return time.Time{}
	}

	return value
}

func parseDateOptional(str string, errs *validation.FieldErrorList) *time.Time {
	if str == "" {
		return nil
	}

	value, err := time.Parse(time.RFC3339, str)
	if err != nil {
		errs.AddError("invalid", err.Error())
		return nil
	}

	return &value
}
