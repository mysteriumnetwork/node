/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package feedback

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserReport_Validate(t *testing.T) {
	for _, data := range []struct {
		email        string
		description  string
		expectToFail bool
		testMSG      string
	}{
		{
			email:        "bad email",
			description:  "A proper description that has no trailing spaces to pad out it's length and has some grain of meaning",
			expectToFail: true,
			testMSG:      "fail bad email",
		},
		{
			email:        "",
			description:  "A proper description that has no trailing spaces to pad out it's length and has some grain of meaning",
			expectToFail: true,
			testMSG:      "fail missing email",
		},
		{
			email:        "   ",
			description:  "A proper description that has no trailing spaces to pad out it's length and has some grain of meaning",
			expectToFail: true,
			testMSG:      "fail missing email 2",
		},
		{
			email:        "qa@qa.qa",
			description:  "A proper description that has no trailing spaces to pad out it's length and has some grain of meaning",
			expectToFail: false,
			testMSG:      "pass valid report",
		},
		{
			email:        "qa@qa.qa",
			description:  "Bad",
			expectToFail: true,
			testMSG:      "fail short description",
		},
		{
			email:        "qa@qa.qa",
			description:  "                                                                                                               ",
			expectToFail: true,
			testMSG:      "fail empty description",
		},
		{
			email:        "      test@email.com        ",
			description:  "     A proper description that has no trailing spaces to pad out it's length and has some grain of meaning              ",
			expectToFail: false,
			testMSG:      "pass valid email and description with spaces",
		},
	} {
		t.Run(data.testMSG, func(t *testing.T) {
			r := UserReport{
				BugReport: BugReport{
					Email:       data.email,
					Description: data.description,
				},
			}

			assert.Equal(t, data.expectToFail, r.Validate() != nil, fmt.Sprintf("email= %s description= %s expected=%t", data.email, data.description, data.expectToFail))
			assert.Equal(t, data.expectToFail, r.BugReport.Validate() != nil, fmt.Sprintf("email= %s description= %s expected=%t", data.email, data.description, data.expectToFail))
		})
	}
}
