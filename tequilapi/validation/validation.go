package validation

import (
	"encoding/json"
)

type FieldError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

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
