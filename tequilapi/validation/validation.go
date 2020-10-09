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

package validation

import (
	"encoding/json"
)

// FieldError structure is produced by validator
type FieldError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// FieldErrorList contains list of FieldError
type FieldErrorList struct {
	list []FieldError
}

// Add adds specified error to error field
func (fel *FieldErrorList) Add(error *FieldError) {
	if error == nil {
		return
	}
	fel.list = append(fel.list, *error)
}

// AddError adds error to error field list with specified code and message
func (fel *FieldErrorList) AddError(code string, message string) {
	fel.Add(&FieldError{code, message})
}

// Required convenience function
func (fel *FieldErrorList) Required() {
	fel.Add(&FieldError{"required", "Field is required"})
}

// Invalid convenience function
func (fel *FieldErrorList) Invalid(message string) {
	fel.Add(&FieldError{"invalid", message})
}

// HasErrors return true if at least one field errors exist
func (fel *FieldErrorList) HasErrors() bool {
	return len(fel.list) > 0
}

// MarshalJSON implements JSON marshaller interface to represent error list as JSON
func (fel FieldErrorList) MarshalJSON() ([]byte, error) {
	return json.Marshal(fel.list)
}

// FieldErrorMap represents a map of field name and corresponding list of errors for that field
type FieldErrorMap struct {
	errorMap map[string]*FieldErrorList
}

// NewErrorMap returns new map of field names to error list
func NewErrorMap() *FieldErrorMap {
	return &FieldErrorMap{make(map[string]*FieldErrorList)}
}

// NewSingleErrorMap a convenience constructor for a single error
func NewSingleErrorMap(key string, err *FieldError) *FieldErrorMap {
	em := NewErrorMap()
	em.errorMap[key] = &FieldErrorList{}
	em.errorMap[key].Add(err)
	return em
}

// ForField returns a list of errors for specified field name
func (fem *FieldErrorMap) ForField(key string) *FieldErrorList {
	var fieldErrors *FieldErrorList
	fieldErrors, exist := fem.errorMap[key]
	if !exist {
		fieldErrors = &FieldErrorList{}
		fem.errorMap[key] = fieldErrors
	}
	return fieldErrors
}

// Set sets errors from another error map
func (fem *FieldErrorMap) Set(errors *FieldErrorMap) {
	for key, fieldErrors := range errors.errorMap {
		fem.errorMap[key] = fieldErrors
	}
}

// MarshalJSON implements JSON marshaller interface to represent error map as JSON
func (fem FieldErrorMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(fem.errorMap)
}

// HasErrors return true if at least one error exist for any field
func (fem *FieldErrorMap) HasErrors() bool {
	for _, fieldErrors := range fem.errorMap {
		if fieldErrors.HasErrors() {
			return true
		}
	}
	return false
}
