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

package openvpn

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfigToArguments(t *testing.T) {
	config := Config{}
	config.AddOptions(
		OptionFlag("flag"),
		OptionFlag("spacy flag"),
		OptionParam("value", "1234"),
		OptionParam("very-value", "1234 5678"),
		OptionParam("spacy value", "1234 5678"),
	)

	arguments, err := config.ConfigToArguments()
	assert.Nil(t, err)
	assert.Equal(t,
		[]string{
			"--flag",
			"--spacy", "flag",
			"--value", "1234",
			"--very-value", "1234", "5678",
			"--spacy", "value", "1234", "5678",
		},
		arguments,
	)
}
