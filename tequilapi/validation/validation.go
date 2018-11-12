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

func (fel *FieldErrorList) AddError(code string, message string) {

	fel.list = append(fel.list, FieldError{code, message})
}

func (fel FieldErrorList) MarshalJSON() ([]byte, error) {
	return json.Marshal(fel.list)
}

type FieldErrorMap struct {
	errorMap map[string]*FieldErrorList
}

func NewErrorMap() *FieldErrorMap {
	return &FieldErrorMap{make(map[string]*FieldErrorList)}
}

func (fem *FieldErrorMap) ForField(key string) *FieldErrorList {
	var fieldErrors *FieldErrorList
	fieldErrors, exist := fem.errorMap[key]
	if !exist {
		fieldErrors = &FieldErrorList{}
		fem.errorMap[key] = fieldErrors
	}
	return fieldErrors
}

func (fem FieldErrorMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(fem.errorMap)
}

func (fem *FieldErrorMap) HasErrors() bool {
	return len(fem.errorMap) > 0
}
