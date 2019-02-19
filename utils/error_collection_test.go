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
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var (
	errorNone   error
	errorFirst  = errors.New("First")
	errorSecond = errors.New("Second")
)

func Test_ErrorCollector_Add(t *testing.T) {
	err := ErrorCollection{}
	assert.Len(t, err, 0)

	err = ErrorCollection{}
	err.Add(errorFirst)
	assert.Len(t, err, 1)
	assert.Equal(t, ErrorCollection([]error{errorFirst}), err)

	err = ErrorCollection{}
	err.Add(errorFirst, errorSecond)
	assert.Len(t, err, 2)
	assert.Equal(t, ErrorCollection([]error{errorFirst, errorSecond}), err)

	err = ErrorCollection{}
	err.Add(errorFirst, errorNone, errorSecond)
	assert.Len(t, err, 2)
	assert.Equal(t, ErrorCollection([]error{errorFirst, errorSecond}), err)
}

func Test_ErrorCollector_String(t *testing.T) {
	err := ErrorCollection{}
	assert.Equal(t, "ErrorCollection: ", err.String())

	err = ErrorCollection{}
	err.Add(errorFirst)
	assert.Equal(t, "ErrorCollection: First", err.String())

	err = ErrorCollection{}
	err.Add(errorFirst, errorSecond)
	assert.Equal(t, "ErrorCollection: First, Second", err.String())
}

func Test_ErrorCollector_Stringf(t *testing.T) {
	err := ErrorCollection{}
	assert.Equal(t, "Failed! ", err.Stringf("Failed! %s", ". "))

	err = ErrorCollection{}
	err.Add(errorFirst)
	assert.Equal(t, "Failed! First", err.Stringf("Failed! %s", ". "))

	err = ErrorCollection{}
	err.Add(errorFirst, errorSecond)
	assert.Equal(t, "Failed! First. Second", err.Stringf("Failed! %s", ". "))
}

func Test_ErrorCollector_Error(t *testing.T) {
	err := ErrorCollection{}
	assert.NoError(t, err.Error())

	err = ErrorCollection{}
	err.Add(errorFirst)
	assert.EqualError(t, err.Error(), "ErrorCollection: First")

	err = ErrorCollection{}
	err.Add(errorFirst, errorSecond)
	assert.EqualError(t, err.Error(), "ErrorCollection: First, Second")
}

func Test_ErrorCollector_Errorf(t *testing.T) {
	err := ErrorCollection{}
	assert.NoError(t, err.Errorf("Failed! %s", ". "))

	err = ErrorCollection{}
	err.Add(errorFirst)
	assert.EqualError(t, err.Errorf("Failed! %s", ". "), "Failed! First")

	err = ErrorCollection{}
	err.Add(errorFirst, errorSecond)
	assert.EqualError(t, err.Errorf("Failed! %s", ". "), "Failed! First. Second")
}
