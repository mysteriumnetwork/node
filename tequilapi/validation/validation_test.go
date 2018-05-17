/*
 * Copyright (C) 2018 The Mysterium Network Authors
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
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestErrorsListRenderedInJSON(t *testing.T) {
	errorMap := NewErrorMap()
	errorMap.ForField("email").AddError("required", "field required")
	errorMap.ForField("email").AddError("another", "error")
	errorMap.ForField("username").AddError("invalid", "field invalid")

	v, err := json.Marshal(errorMap)
	assert.Nil(t, err)

	assert.JSONEq(
		t,
		`{
			"email" : [
				{ "code" : "required" , "message" : "field required" },
				{ "code" : "another" , "message" : "error"}
			],
			"username" : [
				{ "code" : "invalid" , "message" : "field invalid" }
			]
		}`,
		string(v),
	)

}
