/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package utils

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorCollection knows how to combine multiple error strings
type ErrorCollection []error

// Add puts given error to collection
func (ec *ErrorCollection) Add(errors ...error) {
	for _, err := range errors {
		if err != nil {
			*ec = append(*ec, err)
		}
	}
}

// String concatenates collection to single string
func (ec *ErrorCollection) String() string {
	return ec.Stringf("ErrorCollection: %s", ", ")
}

// Stringf returns a string representation of the underlying errors with the given format
func (ec *ErrorCollection) Stringf(format, errorDelimiter string) string {
	errorStrings := make([]string, 0)
	for _, err := range *ec {
		errorStrings = append(errorStrings, err.Error())
	}

	return fmt.Sprintf(format, strings.Join(errorStrings, errorDelimiter))
}

// Error converts collection to single error
func (ec *ErrorCollection) Error() error {
	if len(*ec) == 0 {
		return nil
	}
	return errors.New(ec.String())
}

// Errorf converts collection to single error by wanted format
func (ec *ErrorCollection) Errorf(format, errorDelimiter string) error {
	if len(*ec) == 0 {
		return nil
	}
	return errors.New(ec.Stringf(format, errorDelimiter))
}
